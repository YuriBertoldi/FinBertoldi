# Skill: Contexto FinBertoldi

Voce esta trabalhando no **FinBertoldi** — sistema de controle financeiro familiar.

## Projeto
- **Stack:** Go 1.24 · net/http · html/template · PostgreSQL 16 · HTMX 1.9.12 · Chart.js · Pico CSS v2 · excelize/v2 · ofxgo
- **Modulo Go:** `fincontrol`
- **Local:** `C:/Go/finBertoldi`
- **Containers:** app (8080), postgres (5432), pluggy-service (8081)

## Producao
Dados de acesso ao servidor (IP, SSH, credenciais) estao em `PRODUCAO.md` (ignorado pelo git).
Comandos de deploy e backup estao la tambem.

## Estrutura de pacotes
```
internal/
  models/   — structs de dominio + page data
  store/    — queries PostgreSQL + RunMigrations() (v1-v12) + EnsureAdmin()
  auth/     — sessoes, bcrypt, middlewares
  handler/  — handlers HTTP, InitTemplates(), reports.go, importexport.go, bankimport.go
main.go     — wiring de rotas
pluggy-service/ — microservico de integracao bancaria Pluggy
```

## Arquivos principais
| Arquivo | Responsabilidade |
|---------|-----------------|
| `main.go` | Inicializacao, todas as rotas |
| `internal/models/models.go` | Structs de dominio e page data |
| `internal/store/store.go` | Queries SQL + RunMigrations() |
| `internal/auth/auth.go` | Middlewares Protected/AdminOnly/FamilyAdminOnly/ScreenProtected |
| `internal/handler/handler.go` | Handlers HTTP + InitTemplates() + helpers + Pluggy admin |
| `internal/handler/bankimport.go` | Parsers CSV/OFX + handlers transacoes bancarias |
| `internal/handler/reports.go` | Handlers de relatorio PDF (5 telas) |
| `internal/handler/importexport.go` | Import/Export XLSX via excelize |
| `static/app.css` | Design system completo |
| `templates/base.html` | Layout + sidebar + JS utilitarios |

## Multi-tenancy
Todos os dados isolados por `family_id`. Toda query inclui `WHERE family_id = $X`.

## Migracao (RunMigrations)
Idempotente, roda no startup. 12 migrations.
Tabelas: `families` → `users` → `sessions` → `categorias` → `cartoes` → `despesas_fixas` → `despesas_fixas_mes` → `parcelamentos` → `receitas` → `investimentos` → `reserva_emergencia` → `emprestimos` → `emprestimos_pagamentos` → `screen_permissions` → `transacoes_banco` → `pluggy_items` → `integracoes_config`

## Grupos de categoria (dashboard)
- `basica` → Contas Basicas
- `cartao` → Cartao de Credito
- `vr` → VR/VA

## Funcionalidades
- Dashboard (receitas x despesas, historico 6 meses)
- Despesas fixas + parcelamentos com toggle pago/mes
- Receitas (recorrentes e unicas)
- Investimentos por tipo + reserva de emergencia
- Emprestimos com quitacao parcial
- Planejamento FIRE (taxa poupanca, projecao juros compostos)
- Transacoes bancarias (importacao CSV/OFX, integracao Pluggy)
- Cadastros (categorias + cartoes + integracao bancaria)
- Minha Familia (membros + controle de acesso por tela)
- Usuarios / Familias (admin global)
- Relatorios PDF — 5 telas
- Importar / Exportar XLSX — `/dados` (multi-aba: 5 tipos de dados)
- Tema dark/light + ocultar valores

## Padroes de codigo
- Handlers: `HandleX(db *sql.DB) http.HandlerFunc`
- Template helpers: `brl` (HTML span), `date`, `mesDisplay`, `sub`, `pct`, `progress`, `seq`, `json`, `contains`
- CSS: variaveis em `:root`; `.page-actions` para agrupar botoes no page-header
- Responsivo: tabelas → scroll; sidebar → overlay em <=768px

## Testes
```bash
go test ./...                          # unit (sem banco)
go test -tags=integration ./...        # integracao (precisa postgres)
```
Arquivos: `internal/handler/handler_test.go`, `internal/store/store_test.go`, `internal/store/integration_test.go`, `internal/auth/auth_test.go`

## Documentacao
Documentacao detalhada por modulo em `documentacao/`. Swagger em `documentacao/swagger.yaml`.
