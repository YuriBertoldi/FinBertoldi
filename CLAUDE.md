# FinBertoldi

Sistema de controle financeiro familiar em Go 1.24 + PostgreSQL 16 + HTMX.

## Stack
- Go 1.24 · net/http · html/template
- PostgreSQL 16 · lib/pq
- HTMX 1.9.12 · Chart.js 4.4.1 · Pico CSS v2 (green) · CSS puro (dark/glassmorphism)
- excelize/v2 (geração/leitura de XLSX)
- Docker Compose

## Estrutura de pacotes
- `main.go` — wiring: rotas + startup
- `internal/models/` — structs de domínio e page data
- `internal/store/` — queries PostgreSQL + RunMigrations()
- `internal/auth/` — sessões, bcrypt, middlewares (Protected, AdminOnly, FamilyAdminOnly, ScreenProtected)
- `internal/handler/` — handlers HTTP + InitTemplates() + reports.go + importexport.go
- `static/app.css` — design system
- `templates/base.html` — layout + sidebar mobile

## Produção
Oracle Cloud Always Free ARM · IP `157.151.131.79`
SSH: `ssh -i C:/Go/finBertoldi/Oracle/ssh-key-2026-05-22.key ubuntu@157.151.131.79`
App dir no servidor: `/home/ubuntu/finBertoldi`

## Atualizar produção
```powershell
# Copiar arquivos e rebuildar
scp -i C:/Go/finBertoldi/Oracle/ssh-key-2026-05-22.key -r C:/Go/finBertoldi ubuntu@157.151.131.79:/home/ubuntu/
ssh -i C:/Go/finBertoldi/Oracle/ssh-key-2026-05-22.key ubuntu@157.151.131.79 "cd /home/ubuntu/finBertoldi && docker compose -f docker-compose.prod.yml --env-file .env.prod up -d --build"
```

## Rotas principais
| Método | Rota | Handler |
|--------|------|---------|
| GET | `/` | Dashboard (requer auth) |
| GET/POST | `/login` | Login |
| GET | `/despesas` | Despesas fixas + parcelamentos |
| GET | `/receitas` | Receitas |
| GET | `/investimentos` | Investimentos + reserva EM |
| GET | `/emprestimos` | Empréstimos |
| GET | `/planejamento` | Planejamento FIRE |
| GET | `/cadastros` | Categorias + cartões |
| GET | `/dados` | Importar / Exportar XLSX |
| GET | `/dados/exportar` | Download XLSX com dados atuais |
| GET | `/dados/modelo` | Download modelo em branco |
| POST | `/dados/importar` | Upload XLSX para importação |
| GET | `/relatorio/dashboard` | Relatório PDF do dashboard |
| GET | `/relatorio/despesas` | Relatório PDF de despesas |
| GET | `/relatorio/receitas` | Relatório PDF de receitas |
| GET | `/relatorio/investimentos` | Relatório PDF de investimentos |
| GET | `/relatorio/emprestimos` | Relatório PDF de empréstimos |

Use `/finbertoldi` para contexto completo.
