# Modulo: auth

**Pacote:** `internal/auth`
**Arquivo:** `auth.go`

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
