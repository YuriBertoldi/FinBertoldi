# FinBertoldi

Sistema de controle financeiro familiar em Go 1.24 + PostgreSQL 16 + HTMX.

Este arquivo e a memoria operacional do Codex para este repositorio.

## Stack
- Go 1.24 · net/http · html/template
- PostgreSQL 16 · lib/pq
- HTMX 1.9.12 · Chart.js 4.4.1 · Pico CSS v2 (green) · CSS puro (dark/glassmorphism)
- excelize/v2 para geracao/leitura de XLSX
- ofxgo para parser OFX
- Docker Compose (3 containers: app, postgres, pluggy-service)
- Modulo Go: `fincontrol`

## Estrutura de pacotes
- `main.go` — wiring: rotas + startup
- `internal/models/` — structs de dominio e page data
- `internal/store/` — queries PostgreSQL + RunMigrations() (v1-v12) + EnsureAdmin()
- `internal/auth/` — sessoes, bcrypt, middlewares (Protected, AdminOnly, FamilyAdminOnly, ScreenProtected)
- `internal/handler/` — handlers HTTP + InitTemplates() + helpers + reports.go + importexport.go + bankimport.go
- `static/app.css` — design system
- `templates/base.html` — layout + sidebar mobile + JS utilitarios
- `pluggy-service/` — microservico de integracao bancaria Pluggy

## Arquivos principais
| Arquivo | Responsabilidade |
|---------|-----------------|
| `main.go` | Inicializacao e todas as rotas |
| `internal/models/models.go` | Structs de dominio e page data |
| `internal/store/store.go` | Queries SQL + RunMigrations() |
| `internal/auth/auth.go` | Middlewares Protected/AdminOnly/FamilyAdminOnly/ScreenProtected |
| `internal/handler/handler.go` | Handlers HTTP + InitTemplates() + helpers + Pluggy admin |
| `internal/handler/bankimport.go` | Parsers CSV/OFX + handlers transacoes bancarias |
| `internal/handler/reports.go` | Handlers de relatorio PDF das telas principais |
| `internal/handler/importexport.go` | Import/export XLSX via excelize |
| `static/app.css` | Design system completo |
| `templates/base.html` | Layout + sidebar + JS utilitarios |

## Producao
Dados de acesso ao servidor (IP, SSH key, credenciais) estao em `PRODUCAO.md` (ignorado pelo git).

## Rotas principais
| Metodo | Rota | Handler |
|--------|------|---------|
| GET | `/` | Dashboard, requer auth |
| GET/POST | `/login` | Login |
| GET | `/despesas` | Despesas fixas + parcelamentos |
| GET | `/receitas` | Receitas |
| GET | `/pagamentos` | Pagamentos do mes |
| GET | `/investimentos` | Investimentos + reserva EM |
| GET | `/emprestimos` | Emprestimos |
| GET | `/planejamento` | Planejamento FIRE |
| GET | `/cadastros` | Categorias + cartoes + integracao bancaria (admin) |
| GET | `/transacoes` | Transacoes bancarias (CSV/OFX/Pluggy) |
| GET | `/dados` | Importar / Exportar XLSX |
| GET | `/relatorio/*` | Relatorios PDF (5 telas) |

## Funcionalidades
- Dashboard com receitas x despesas e historico de 6 meses
- Despesas fixas + parcelamentos com toggle pago/mes
- Receitas recorrentes e unicas
- Investimentos por tipo + reserva de emergencia
- Emprestimos com quitacao parcial
- Planejamento FIRE com taxa de poupanca e projecao
- Transacoes bancarias: importacao CSV/OFX, integracao Pluggy
- Cadastros de categorias, cartoes, integracao bancaria
- Minha Familia: membros + controle de acesso por tela
- Usuarios / Familias para admin global
- Relatorios PDF em 5 telas
- Importar / Exportar XLSX multi-aba
- Tema dark/light + ocultar valores

## Multi-tenancy
Todos os dados isolados por `family_id`. Toda query inclui `WHERE family_id = $X`.

## Migracoes
`RunMigrations()` e idempotente e roda no startup. 12 migrations.

Tabelas:
`families` → `users` → `sessions` → `categorias` → `cartoes` → `despesas_fixas` → `despesas_fixas_mes` → `parcelamentos` → `receitas` → `investimentos` → `reserva_emergencia` → `emprestimos` → `emprestimos_pagamentos` → `screen_permissions` → `transacoes_banco` → `pluggy_items` → `integracoes_config`

## Padroes de codigo
- Handlers seguem o formato `HandleX(db *sql.DB) http.HandlerFunc`
- Template helpers: `brl`, `date`, `mesDisplay`, `sub`, `pct`, `progress`, `seq`, `json`, `contains`
- CSS usa variaveis em `:root`
- Responsivo: tabelas viram scroll; sidebar vira overlay em `<=768px`

## Testes
```bash
go test ./...                          # unitarios, sem banco
go test -tags=integration ./...        # integracao, precisa PostgreSQL
```

## Documentacao
Documentacao detalhada por modulo em `documentacao/`. Swagger em `documentacao/swagger.yaml`.
