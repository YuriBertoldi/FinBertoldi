# Skill: Contexto FinBertoldi

Você está trabalhando no **FinBertoldi** — sistema de controle financeiro familiar.

## Projeto
- **Stack:** Go 1.24 · net/http · html/template · PostgreSQL 16 · HTMX 1.9.12 · Chart.js · Pico CSS v2 · excelize/v2
- **Módulo Go:** `fincontrol`
- **Local:** `C:/Go/finBertoldi`

## Produção (Oracle Cloud Always Free ARM)
- **IP:** `157.151.131.79` → http://157.151.131.79
- **SSH:** `ssh -i C:/Go/finBertoldi/Oracle/ssh-key-2026-05-22.key ubuntu@157.151.131.79`
- **App dir:** `/home/ubuntu/finBertoldi`
- **Rebuild:** `docker compose -f docker-compose.prod.yml --env-file .env.prod up -d --build`
- **Logs:** `docker logs finbertoldi-app-1 --tail 50`
- **DB:** `docker exec -it finbertoldi-postgres-1 psql -U fincontrol -d fincontrol`
- **Admin padrão:** `admin@finbertoldi.com` / `admin123`

## Enviar para produção
```powershell
# Copiar tudo e rebuildar
scp -i C:/Go/finBertoldi/Oracle/ssh-key-2026-05-22.key -r C:/Go/finBertoldi ubuntu@157.151.131.79:/home/ubuntu/
ssh -i C:/Go/finBertoldi/Oracle/ssh-key-2026-05-22.key ubuntu@157.151.131.79 "cd /home/ubuntu/finBertoldi && docker compose -f docker-compose.prod.yml --env-file .env.prod up -d --build"

# Só CSS (sem rebuild)
scp -i C:/Go/finBertoldi/Oracle/ssh-key-2026-05-22.key C:/Go/finBertoldi/static/app.css ubuntu@157.151.131.79:/home/ubuntu/finBertoldi/static/app.css
```

## Estrutura de pacotes
```
internal/
  models/   — structs de domínio + page data
  store/    — queries PostgreSQL + RunMigrations() + EnsureAdmin()
  auth/     — sessões, bcrypt, middlewares
  handler/  — handlers HTTP, InitTemplates(), reports.go, importexport.go
main.go     — wiring de rotas
```

## Arquivos principais
| Arquivo | Responsabilidade |
|---------|-----------------|
| `main.go` | Inicialização, todas as rotas |
| `internal/models/models.go` | Structs de domínio e page data |
| `internal/store/store.go` | Queries SQL + RunMigrations() |
| `internal/auth/auth.go` | Middlewares Protected/AdminOnly/FamilyAdminOnly/ScreenProtected |
| `internal/handler/handler.go` | Handlers HTTP + InitTemplates() + helpers |
| `internal/handler/reports.go` | Handlers de relatório PDF (5 telas) |
| `internal/handler/importexport.go` | Import/Export XLSX via excelize |
| `static/app.css` | Design system completo |
| `templates/base.html` | Layout + sidebar + JS utilitários |

## Multi-tenancy
Todos os dados isolados por `family_id`. Toda query inclui `WHERE family_id = $X`.

## Migração (RunMigrations)
Idempotente, roda no startup. Tabelas:
`families → users → sessions → categorias → cartoes → despesas_fixas → despesas_fixas_mes → parcelamentos → receitas → investimentos → reserva_emergencia → emprestimos → emprestimos_pagamentos → screen_permissions`

## Grupos de categoria (dashboard)
- `basica` → Contas Básicas
- `cartao` → Cartão de Crédito
- `vr` → VR/VA

## Funcionalidades
- Dashboard (receitas × despesas, histórico 6 meses)
- Despesas fixas + parcelamentos com toggle pago/mês
- Receitas (recorrentes e únicas)
- Investimentos por tipo + reserva de emergência
- Empréstimos com quitação parcial
- Planejamento FIRE (taxa poupança, projeção juros compostos)
- Cadastros (categorias + cartões)
- Minha Família (membros + controle de acesso por tela)
- Usuários / Famílias (admin global)
- **Relatórios PDF** — 5 telas, botão "📄 Relatório" em cada página
- **Importar / Exportar XLSX** — `/dados` (multi-aba: 5 tipos de dados)
- Tema dark/light + ocultar valores

## Padrões de código
- Handlers: `HandleX(db *sql.DB) http.HandlerFunc`
- Template helpers: `brl` (HTML span), `date`, `mesDisplay`, `sub`, `pct`, `progress`, `seq`, `json`, `contains`
- CSS: variáveis em `:root`; `.page-actions` para agrupar botões no page-header
- Responsivo: tabelas → scroll; sidebar → overlay em ≤768px

## Testes
```bash
go test ./...                          # unit (sem banco)
go test -tags=integration ./...        # integração (precisa postgres)
```
Arquivos: `internal/handler/handler_test.go`, `internal/store/store_test.go`, `internal/store/integration_test.go`, `internal/auth/auth_test.go`
