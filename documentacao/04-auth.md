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

## Recuperacao de Senha (email.go)

### `SendResetEmail(to, token, baseURL string) error`

Envia email HTML com link de reset de senha via SMTP. Se SMTP nao configurado, loga o link no console (fallback para dev).

**Variaveis de ambiente:**
| Variavel | Descricao | Exemplo |
|----------|-----------|---------|
| `SMTP_HOST` | Servidor SMTP | smtp.gmail.com |
| `SMTP_PORT` | Porta SMTP | 587 |
| `SMTP_USER` | Usuario SMTP | user@gmail.com |
| `SMTP_PASS` | Senha/app password | xxxx |
| `SMTP_FROM` | Remetente | noreply@finbertoldi.com |

### Fluxo de recuperacao

1. Usuario acessa `/forgot-password` e informa email
2. Sistema gera token hex de 64 chars com expiracao de 1h
3. Token salvo em `password_resets` no banco
4. Email enviado com link `/reset-senha?token=XXX`
5. Usuario clica no link, define nova senha
6. Token marcado como usado, senha atualizada com bcrypt
