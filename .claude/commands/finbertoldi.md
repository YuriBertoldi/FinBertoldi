# Skill: Contexto FinBertoldi

Voce esta trabalhando no **FinBertoldi** — sistema de controle financeiro familiar.

## Projeto
- **Stack:** Go 1.27 · net/http · html/template · PostgreSQL 16 · HTMX 1.9.12 · Chart.js · Pico CSS v2 · excelize/v2 · ofxgo
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
  store/    — queries PostgreSQL + RunMigrations() (v1-v21) + EnsureAdmin()
  auth/     — sessoes, bcrypt, middlewares
  handler/  — handlers HTTP, InitTemplates(), reports.go, importexport.go, bankimport.go, whatsapp.go
  contas/   — contas do mes (WhatsApp) + envio e-mail/Telegram + agendador mensal
  integrations/ — BCB, cotacoes, Telegram, BrasilAPI, Google Sheets + scheduler
main.go     — wiring de rotas
pluggy-service/ — microservico de integracao bancaria Pluggy
```

## Arquivos principais
| Arquivo | Responsabilidade |
|---------|-----------------|
| `main.go` | Inicializacao, todas as rotas |
| `internal/models/models.go` | Structs de dominio e page data |
| `internal/store/<dominio>.go` | Queries SQL, um arquivo por dominio (`despesas.go`, `usuarios.go`...); mapa em `documentacao/03-store.md` |
| `internal/store/migrations.go` | Migrations + RunMigrations() |
| `internal/auth/auth.go` | Middlewares Protected/AdminOnly/FamilyAdminOnly/ScreenProtected |
| `internal/handler/<tela>.go` | Um arquivo por tela (`despesas.go`, `receitas.go`, `autenticacao.go`...); mapa em `documentacao/05-handler.md` |
| `internal/handler/templates.go` | InitTemplates() + render |
| `internal/handler/bankimport.go` | Parsers CSV/OFX + handlers transacoes bancarias |
| `internal/handler/reports.go` | Handlers de relatorio PDF (5 telas) |
| `internal/handler/importexport.go` | Import/Export XLSX via excelize |
| `static/app.css` | Design system completo |
| `templates/base.html` | Layout + sidebar + JS utilitarios |

## Multi-tenancy
Todos os dados isolados por `family_id`. Toda query inclui `WHERE family_id = $X`.

## Migracao (RunMigrations)
Idempotente, roda no startup. 21 migrations.
Tabelas: `families` → `users` → `sessions` → `categorias` → `cartoes` → `despesas_fixas` → `despesas_fixas_mes` → `parcelamentos` → `receitas` → `investimentos` → `reserva_emergencia` → `emprestimos` → `emprestimos_pagamentos` → `screen_permissions` → `transacoes_banco` → `pluggy_items` → `integracoes_config` → `password_resets` → `dados_economicos` → `feriados` → `integracoes_kv` → `dashboard_widgets` → `user_notificacoes`

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
- Contas do mes: WhatsApp, e-mail e Telegram; envio mensal em /notificacoes
- Integracoes: BCB, cotacoes, feriados, Telegram, Google Sheets
- Dashboard com widgets reordenaveis (ordem e coluna)
- Importar / Exportar XLSX — `/dados` (multi-aba: 5 tipos de dados)
- Tema dark/light + ocultar valores

## Padroes de codigo
- Handlers: `HandleX(db *sql.DB) http.HandlerFunc`
- Template helpers: `brl` (HTML span), `date`, `mesDisplay`, `sub`, `pct`, `progress`, `seq`, `json`, `contains`, `assetV`, `dict`
- CSS: variaveis em `:root`; `.page-actions` para agrupar botoes no page-header
- Responsivo: tabelas `.table-responsive` → cartoes em <=640px; sidebar → overlay em <=960px; fontes via `--fs-*`

## Testes
```bash
go test ./...                          # unit (sem banco)
go test -tags=integration ./...        # integracao (precisa postgres)
```
Push na `main` roda CI (unit + integracao com PostgreSQL) e faz deploy automatico; migrations aplicam no startup.
Arquivos: `internal/*/*_test.go`; integracao em `internal/store/*integration_test.go` e `internal/integrations/sheets_integration_test.go` (ver `TESTES.md`)

## Documentacao
Documentacao detalhada por modulo em `documentacao/`. Swagger em `documentacao/swagger.yaml`.
