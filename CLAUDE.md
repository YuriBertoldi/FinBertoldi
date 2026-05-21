# FinBertoldi

Sistema de controle financeiro familiar em Go 1.22 + PostgreSQL 16 + HTMX.

## Stack
- Go 1.22 · net/http · html/template
- PostgreSQL 16 · lib/pq
- HTMX 1.9.12 · Chart.js 4.4.1 · CSS puro (dark/glassmorphism)
- Docker Compose

## Estrutura de pacotes
- `main.go` — wiring: rotas + startup
- `internal/models/` — structs de domínio e page data
- `internal/store/` — queries PostgreSQL + RunMigrations()
- `internal/auth/` — sessões, bcrypt, middlewares (Protected, AdminOnly, etc.)
- `internal/handler/` — handlers HTTP + InitTemplates()
- `static/app.css` — design system
- `templates/base.html` — layout + sidebar mobile

## Produção
Oracle Cloud Always Free ARM · IP `141.148.34.13`  
SSH: `ssh -i ~/Downloads/ssh-key-2026-05-19.key ubuntu@141.148.34.13`

Use `/finbertoldi` para contexto completo.
