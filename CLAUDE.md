# FinBertoldi

Sistema de controle financeiro familiar em Go 1.27 + PostgreSQL 16 + HTMX.

## Stack
- Go 1.27 · net/http · html/template
- PostgreSQL 16 · lib/pq
- HTMX 1.9.12 · Chart.js 4.4.1 · Pico CSS v2 (green) · CSS puro (dark/glassmorphism)
- excelize/v2 (geracao/leitura de XLSX)
- ofxgo (parser OFX para importacao bancaria)
- Docker Compose (3 containers: app, postgres, pluggy-service)
- Caddy (reverse proxy HTTPS em producao)

## Estrutura de pacotes
- `main.go` — wiring: rotas + startup
- `internal/models/` — structs de dominio e page data
- `internal/store/` — queries PostgreSQL, um arquivo por dominio (`despesas.go`, `usuarios.go`...); `migrations.go` com RunMigrations() (22 migrations); mapa em `documentacao/03-store.md`
- `internal/auth/` — sessoes, bcrypt, middlewares (Protected, AdminOnly, FamilyAdminOnly, ScreenProtected) + email.go (SMTP)
- `internal/handler/` — handlers HTTP, um arquivo por tela (`despesas.go`, `receitas.go`, `autenticacao.go`...); `templates.go` (InitTemplates/render); mapa em `documentacao/05-handler.md`
- `internal/integrations/` — integracoes gratuitas (BCB, cotacoes, Telegram, BrasilAPI, Google Sheets) + scheduler
- `internal/contas/` — resumo das contas do mes (WhatsApp) + envio por e-mail/Telegram + agendador mensal
- `static/app.css` — design system
- `templates/` — templates HTML (kebab-case para nomes compostos)
- `pluggy-service/` — microservico de integracao bancaria Pluggy (container separado)

## Producao
Dados de acesso ao servidor de producao estao em `PRODUCAO.md` (ignorado pelo git).

## Rotas principais
| Metodo | Rota | Handler |
|--------|------|---------|
| GET | `/healthz` | Saude: 200 se o banco responde, 503 se nao (publica) |
| GET | `/` | Dashboard (requer auth) |
| GET/POST | `/login` | Login |
| GET/POST | `/forgot-password` | Recuperacao de senha |
| GET/POST | `/reset-senha` | Redefinir senha via token |
| GET | `/despesas` | Despesas fixas + parcelamentos |
| GET | `/receitas` | Receitas |
| GET | `/pagamentos` | Pagamentos do mes (+ janela WhatsApp / envio Telegram e e-mail) |
| GET/POST | `/notificacoes` | Envio mensal das contas por usuario |
| GET | `/investimentos` | Investimentos + reserva EM |
| GET | `/emprestimos` | Emprestimos |
| GET | `/planejamento` | Planejamento FIRE |
| GET | `/cadastros` | Categorias + cartoes + integracao bancaria (admin) |
| GET | `/transacoes` | Transacoes bancarias (CSV/OFX/Pluggy) + conciliacao |
| GET | `/integracoes` | Integracoes gratuitas (BCB, cotacoes, Telegram, etc) |
| GET | `/dados` | Importar / Exportar XLSX |
| GET | `/relatorio/*` | Relatorios PDF (dashboard, despesas, receitas, investimentos, emprestimos) |

## Pluggy Service (microservico)
Container separado em `pluggy-service/`. Comunica com API Pluggy para sincronizar transacoes bancarias.
- Porta: 8081
- Endpoints: `/api/pluggy/status`, `/api/pluggy/connect-token`, `/api/pluggy/webhook`, `/api/pluggy/items`, `/api/pluggy/sync/{item_id}`
- Credenciais: le de `integracoes_config` no banco ou env vars `PLUGGY_CLIENT_ID`/`PLUGGY_CLIENT_SECRET`
- Sync automatico configuravel via `SYNC_INTERVAL` (default: 6h)

## Integracoes gratuitas
Ativadas por familia via `/integracoes`. Scheduler em background coleta dados:
- **BCB** (Selic/CDI/IPCA) — a cada 6h
- **AwesomeAPI** (USD/EUR/BTC) — a cada 6h
- **BrasilAPI** (feriados) — a cada 6h
- **Telegram** (alertas de vencimento) — a cada 24h
- **Google Sheets** — Service Account; abas Resumo/Contas/Receitas/Parcelamentos/Investimentos/Transacoes, 1x/dia

## Tabelas do banco (22 migrations)
`families` → `users` → `sessions` → `categorias` → `cartoes` → `despesas_fixas` → `despesas_fixas_mes` → `parcelamentos` → `parcelamentos_mes` → `receitas` → `investimentos` → `reserva_emergencia` → `emprestimos` → `emprestimos_pagamentos` → `screen_permissions` → `transacoes_banco` → `pluggy_items` → `integracoes_config` → `password_resets` → `dados_economicos` → `feriados` → `integracoes_kv` → `dashboard_widgets` → `user_notificacoes`

## Documentacao
Documentacao detalhada por modulo em `documentacao/`. Swagger da API em `documentacao/swagger.yaml`.

Use `/finbertoldi` para contexto completo.

## Testes
- `go test ./...` (unit) · `go test -tags=integration ./...` (PostgreSQL; roda no CI com `INTEGRATION_REQUIRED=1`)
- PR e push na `main` rodam CI: build, `gofmt`, `go vet`, `staticcheck`, `govulncheck` e testes (app e pluggy-service). Deploy so manual, por *Run workflow* na `main` (GitHub Actions), com backup `pg_dump` antes; migrations aplicam no startup do app.
