# Modulo: auth

**Pacote:** `internal/auth`
**Arquivos:** `auth.go`, `email.go`

## Responsabilidade

Autenticacao e autorizacao do sistema. Gerencia sessoes, cookies e middlewares de protecao de rotas.

## Sessoes

- Token aleatorio de 64 caracteres hexadecimais
- Armazenado em cookie HTTP-only com SameSite=Lax
- Expiracao: 30 dias
- Tabela `sessions` no banco (token, user_id, expires_at)

## Funcoes publicas

| Funcao | Descricao |
|--------|-----------|
| `GenerateToken()` | Gera token aleatorio de 64 chars hex |
| `SetSessionCookie(w, token, expiry)` | Define cookie de sessao no response |
| `ClearSessionCookie(w)` | Remove cookie de sessao |
| `CurrentUser(r)` | Retorna `*models.User` do contexto (injetado pelo middleware) |
| `AuthUser(db, r)` | Le cookie e retorna usuario autenticado (ou nil) |

## Middlewares

### `Protected(db, handler)`
- Exige sessao valida
- Redireciona para `/login` se nao autenticado ou usuario inativo
- Injeta `*models.User` no contexto da request via `context.WithValue`
- Todos os demais middlewares encadeiam sobre este

### `AdminOnly(db, handler)`
- Encadeia sobre `Protected`
- Exige `user.Admin == true`
- Retorna HTTP 403 se nao for admin global
- Usado em: `/usuarios`, `/familias`, `/cadastros/pluggy/*`

### `FamilyAdminOnly(db, handler)`
- Encadeia sobre `Protected`
- Exige `user.FamilyAdmin == true` OU `user.Admin == true`
- Retorna HTTP 403 se nao for admin de familia
- Usado em: `/minha-familia`

### `ScreenProtected(db, screen, handler)`
- Encadeia sobre `Protected`
- Admin e FamilyAdmin sempre passam
- Para usuarios comuns, verifica se a tela (`screen`) esta na lista de telas bloqueadas
- Se bloqueada, renderiza pagina HTML com mensagem "Acesso bloqueado"
- Usado em: `/despesas`, `/receitas`, `/investimentos`, `/emprestimos`, `/cadastros`, `/transacoes`, etc

## Hierarquia de acesso

```
Admin Global
  └── Acesso total (todas as telas + administracao)

FamilyAdmin
  └── Acesso a telas da familia + gerenciamento de membros

Usuario Comum
  └── Acesso apenas a telas liberadas pelo FamilyAdmin
```

## Testes

`auth_test.go` — testa middlewares, geracao de token, manipulacao de cookies

## E-mail (email.go)

### `SendEmail(to, subject, htmlBody string) error`

Envio de e-mail HTML usado pela recuperacao de senha e pelas contas do mes. Porta 465 usa TLS direto; as demais, STARTTLS. O assunto e codificado em UTF-8 e o remetente pode ter nome (`FinBertoldi <x@gmail.com>`).

### Configuracao embutida para Gmail

Para Gmail basta `SMTP_USER` + `SMTP_PASS` (senha de app de 16 letras; espacos sao removidos). O resto tem padrao no codigo (`lerSMTP`):

| Variavel | Padrao | Descricao |
|----------|--------|-----------|
| `SMTP_USER` | — | Conta que envia (ex.: `finbertoldi.app@gmail.com`) |
| `SMTP_PASS` | — | Senha de app do Google (requer verificacao em duas etapas) |
| `SMTP_HOST` | `smtp.gmail.com` se o usuario for `@gmail.com` | Outros provedores (Brevo, Outlook...) |
| `SMTP_PORT` | `587` | `465` = TLS direto |
| `SMTP_FROM` | `FinBertoldi <SMTP_USER>` | Remetente |

`SMTPConfigurado()` indica se ha servidor e remetente. Em producao as variaveis ficam no `.env.prod` e o `docker-compose.prod.yml` as repassa ao container `app`.

## Recuperacao de Senha

### `SendResetEmail(to, token, baseURL string) error`

Envia o link de reset via `SendEmail`. Se o SMTP nao estiver configurado, loga o link no console (fallback para dev).

### Fluxo de recuperacao

1. Usuario acessa `/forgot-password` e informa email
2. Sistema gera token hex de 64 chars com expiracao de 1h
3. Token salvo em `password_resets` no banco
4. Email enviado com link `/reset-senha?token=XXX`
5. Usuario clica no link, define nova senha
6. Token marcado como usado, senha atualizada com bcrypt e **todas as sessoes do usuario sao encerradas**

## Troca de senha e sessoes

- `/minha-senha` exige a **senha atual**, **minimo de 8 caracteres**, senha diferente da atual e recusa senhas conhecidas (`admin123`, `12345678`, `changeme`) — `validarNovaSenha` em `handler.go`.
- Toda troca/redefinicao chama `store.TrocarSenha`, que grava o hash e apaga as sessoes do usuario em outros aparelhos (quem troca a propria senha continua logado; redefinicao por admin ou por e-mail encerra todas).
- O admin inicial (`EnsureAdmin`) usa `ADMIN_PASSWORD` ou uma senha aleatoria exibida uma unica vez no log — nunca uma senha fixa.
