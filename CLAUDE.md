# FinBertoldi

Sistema de controle financeiro familiar em Go 1.24 + PostgreSQL 16 + HTMX.

## Stack
- Go 1.24 · net/http · html/template
- PostgreSQL 16 · lib/pq
- HTMX 1.9.12 · Chart.js 4.4.1 · Pico CSS v2 (green) · CSS puro (dark/glassmorphism)
- excelize/v2 (geracao/leitura de XLSX)
- ofxgo (parser OFX para importacao bancaria)
- Docker Compose (3 containers: app, postgres, pluggy-service)

## Estrutura de pacotes
- `main.go` — wiring: rotas + startup
- `internal/models/` — structs de dominio e page data
- `internal/store/` — queries PostgreSQL + RunMigrations() (12 migrations)
- `internal/auth/` — sessoes, bcrypt, middlewares (Protected, AdminOnly, FamilyAdminOnly, ScreenProtected)
- `internal/handler/` — handlers HTTP + InitTemplates() + reports.go + importexport.go + bankimport.go
- `static/app.css` — design system
- `templates/base.html` — layout + sidebar mobile
- `pluggy-service/` — microservico de integracao bancaria Pluggy (container separado)

## Producao
Dados de acesso ao servidor de producao estao em `PRODUCAO.md` (ignorado pelo git).

## Rotas principais
| Metodo | Rota | Handler |
|--------|------|---------|
| GET | `/` | Dashboard (requer auth) |
| GET/POST | `/login` | Login |
| GET | `/despesas` | Despesas fixas + parcelamentos |
| GET | `/receitas` | Receitas |
| GET | `/pagamentos` | Pagamentos do mes |
| GET | `/investimentos` | Investimentos + reserva EM |
| GET | `/emprestimos` | Emprestimos |
| GET | `/planejamento` | Planejamento FIRE |
| GET | `/cadastros` | Categorias + cartoes + integracao bancaria (admin) |
| GET | `/transacoes` | Transacoes bancarias (CSV/OFX/Pluggy) |
| POST | `/transacoes/import-csv` | Importar CSV bancario |
| POST | `/transacoes/import-ofx` | Importar OFX bancario |
| POST | `/cadastros/pluggy/config` | Salvar config Pluggy (admin) |
| POST | `/cadastros/pluggy/connect-token` | Gerar token Connect Widget (admin) |
| POST | `/cadastros/pluggy/items` | Registrar item Pluggy (admin) |
| POST | `/cadastros/pluggy/{item_id}/sync` | Forcar sync Pluggy (admin) |
| POST | `/cadastros/pluggy/{item_id}/disconnect` | Desconectar banco (admin) |
| GET | `/dados` | Importar / Exportar XLSX |
| GET | `/dados/exportar` | Download XLSX com dados atuais |
| GET | `/dados/modelo` | Download modelo em branco |
| POST | `/dados/importar` | Upload XLSX para importacao |
| GET | `/relatorio/dashboard` | Relatorio PDF do dashboard |
| GET | `/relatorio/despesas` | Relatorio PDF de despesas |
| GET | `/relatorio/receitas` | Relatorio PDF de receitas |
| GET | `/relatorio/investimentos` | Relatorio PDF de investimentos |
| GET | `/relatorio/emprestimos` | Relatorio PDF de emprestimos |

## Pluggy Service (microservico)
Container separado em `pluggy-service/`. Comunica com API Pluggy para sincronizar transacoes bancarias.
- Porta: 8081
- Endpoints: `/api/pluggy/status`, `/api/pluggy/connect-token`, `/api/pluggy/webhook`, `/api/pluggy/items`, `/api/pluggy/sync/{item_id}`
- Credenciais: le de `integracoes_config` no banco ou env vars `PLUGGY_CLIENT_ID`/`PLUGGY_CLIENT_SECRET`
- Sync automatico configuravel via `SYNC_INTERVAL` (default: 6h)

## Tabelas do banco (12 migrations)
`families` → `users` → `sessions` → `categorias` → `cartoes` → `despesas_fixas` → `despesas_fixas_mes` → `parcelamentos` → `receitas` → `investimentos` → `reserva_emergencia` → `emprestimos` → `emprestimos_pagamentos` → `screen_permissions` → `transacoes_banco` → `pluggy_items` → `integracoes_config`

## Documentacao
Documentacao detalhada por modulo em `documentacao/`. Swagger da API em `documentacao/swagger.yaml`.

Use `/finbertoldi` para contexto completo.
