# FinBertoldi

Sistema de controle financeiro familiar em Go 1.24 + PostgreSQL 16 + HTMX.

## Stack
- Go 1.24 · net/http · html/template
- PostgreSQL 16 · lib/pq
- HTMX 1.9.12 · Chart.js 4.4.1 · Pico CSS v2 (green) · CSS puro (dark/glassmorphism)
- excelize/v2 (geracao/leitura de XLSX)
- ofxgo (parser OFX para importacao bancaria)
- Docker Compose (3 containers: app, postgres, pluggy-service)
- Caddy (reverse proxy HTTPS em producao)

## Estrutura de pacotes
- `main.go` — wiring: rotas + startup
- `internal/models/` — structs de dominio e page data
- `internal/store/` — queries PostgreSQL + RunMigrations() (15 migrations)
- `internal/auth/` — sessoes, bcrypt, middlewares (Protected, AdminOnly, FamilyAdminOnly, ScreenProtected) + email.go (SMTP)
- `internal/handler/` — handlers HTTP + InitTemplates() + reports.go + importexport.go + bankimport.go
- `internal/integrations/` — integracoes gratuitas (BCB, cotacoes, Telegram, BrasilAPI, Sheets) + scheduler
- `static/app.css` — design system
- `templates/` — templates HTML (kebab-case para nomes compostos)
- `pluggy-service/` — microservico de integracao bancaria Pluggy (container separado)

## Producao
Dados de acesso ao servidor de producao estao em `PRODUCAO.md` (ignorado pelo git).

## Rotas principais
| Metodo | Rota | Handler |
|--------|------|---------|
| GET | `/` | Dashboard (requer auth) |
| GET/POST | `/login` | Login |
| GET/POST | `/forgot-password` | Recuperacao de senha |
| GET/POST | `/reset-senha` | Redefinir senha via token |
| GET | `/despesas` | Despesas fixas + parcelamentos |
| GET | `/receitas` | Receitas |
| GET | `/pagamentos` | Pagamentos do mes |
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
- **Google Sheets** (export) — stub

## Tabelas do banco (15 migrations)
`families` → `users` → `sessions` → `categorias` → `cartoes` → `despesas_fixas` → `despesas_fixas_mes` → `parcelamentos` → `parcelamentos_mes` → `receitas` → `investimentos` → `reserva_emergencia` → `emprestimos` → `emprestimos_pagamentos` → `screen_permissions` → `transacoes_banco` → `pluggy_items` → `integracoes_config` → `password_resets` → `dados_economicos` → `feriados` → `integracoes_kv`

## Documentacao
Documentacao detalhada por modulo em `documentacao/`. Swagger da API em `documentacao/swagger.yaml`.

Use `/finbertoldi` para contexto completo.
