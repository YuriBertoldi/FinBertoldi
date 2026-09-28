# Visao Geral — FinBertoldi

## O que e

Sistema de controle financeiro familiar multi-tenant. Cada familia tem seus dados isolados (despesas, receitas, investimentos, etc). Suporta multiplos usuarios por familia com controle de acesso granular por tela.

## Arquitetura

O sistema e composto por 3 containers Docker:

1. **app** (Go, porta 8080) — aplicacao principal com SSR (server-side rendering) via `html/template` + HTMX
2. **postgres** (PostgreSQL 16, porta 5432) — banco de dados relacional
3. **pluggy-service** (Go, porta 8081) — microservico de integracao bancaria via Pluggy API

### Fluxo de requisicao

```
Browser → App (Go) → PostgreSQL
                   → Pluggy Service → Pluggy API (api.pluggy.ai)
```

O browser faz requests HTTP para o app principal. O app renderiza HTML no servidor e envia de volta. HTMX e usado para atualizacoes parciais da pagina sem full reload.

O pluggy-service e acessado apenas pelo app principal (proxy HTTP interno). Nunca e exposto diretamente ao browser.

## Multi-tenancy

Toda tabela de dados tem coluna `family_id`. Toda query SQL inclui `WHERE family_id = $X`. Um usuario so ve dados da propria familia.

## Autenticacao

Sessoes em cookie HTTP-only com token aleatorio de 64 chars hex. Sessoes armazenadas na tabela `sessions` com expiracao de 30 dias.

## Stack

| Componente | Tecnologia |
|------------|-----------|
| Linguagem | Go 1.24 |
| HTTP Router | net/http (stdlib, Go 1.22+ com path params) |
| Templates | html/template (stdlib) |
| Banco | PostgreSQL 16 via lib/pq |
| Frontend | HTMX 1.9.12 (atualizacoes parciais) |
| Graficos | Chart.js 4.4.1 |
| CSS Base | Pico CSS v2 (green) |
| CSS Custom | app.css com glassmorphism + dark/light theme |
| XLSX | excelize/v2 |
| OFX | ofxgo |
| Integracao Bancaria | Pluggy API |
| Indicadores BCB | API BCB (Selic, CDI, IPCA) |
| Cotacoes | AwesomeAPI (USD, EUR, BTC) |
| Feriados | BrasilAPI |
| Alertas | Telegram Bot API |
| Email | net/smtp (recuperacao de senha) |
| Infra | Docker Compose + Caddy (HTTPS) |

## Modulos do sistema

```
finBertoldi/
├── main.go                          # Wiring: rotas + startup
├── internal/
│   ├── models/models.go             # Structs de dominio e page data
│   ├── store/store.go               # Queries SQL + migrations (v1-v15)
│   ├── auth/
│   │   ├── auth.go                  # Sessoes, bcrypt, middlewares
│   │   └── email.go                 # Envio de email SMTP (reset senha)
│   ├── handler/
│   │   ├── handler.go               # Handlers HTTP principais
│   │   ├── bankimport.go            # Importacao CSV/OFX + conciliacao
│   │   ├── reports.go               # Relatorios PDF
│   │   └── importexport.go          # Import/Export XLSX
│   └── integrations/
│       ├── scheduler.go             # Scheduler periodico (goroutines)
│       ├── bcb.go                   # API Banco Central (Selic/CDI/IPCA)
│       ├── cotacoes.go              # AwesomeAPI (USD/EUR/BTC)
│       ├── telegram.go              # Telegram Bot (alertas)
│       ├── brasilapi.go             # BrasilAPI (feriados)
│       └── sheets.go                # Google Sheets (export)
├── templates/                       # Templates HTML (Go html/template)
├── static/app.css                   # Design system CSS
├── pluggy-service/                  # Microservico integracao bancaria Pluggy
└── documentacao/                    # Documentacao detalhada
```
