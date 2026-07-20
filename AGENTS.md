# FinBertoldi

Sistema de controle financeiro familiar em Go 1.24 + PostgreSQL 16 + HTMX.

Este arquivo é a memória operacional do Codex para este repositório. Ele substitui o contexto que antes ficava em `CLAUDE.md` e em `.claude/commands/finbertoldi.md`.

## Stack
- Go 1.24 · net/http · html/template
- PostgreSQL 16 · lib/pq
- HTMX 1.9.12 · Chart.js 4.4.1 · Pico CSS v2 (green) · CSS puro (dark/glassmorphism)
- excelize/v2 para geração/leitura de XLSX
- Docker Compose
- Módulo Go: `fincontrol`

## Estrutura de pacotes
- `main.go` — wiring: rotas + startup
- `internal/models/` — structs de domínio e page data
- `internal/store/` — queries PostgreSQL + RunMigrations() + EnsureAdmin()
- `internal/auth/` — sessões, bcrypt, middlewares (Protected, AdminOnly, FamilyAdminOnly, ScreenProtected)
- `internal/handler/` — handlers HTTP + InitTemplates() + helpers + reports.go + importexport.go
- `static/app.css` — design system
- `templates/base.html` — layout + sidebar mobile + JS utilitários

## Arquivos principais
| Arquivo | Responsabilidade |
|---------|-----------------|
| `main.go` | Inicialização e todas as rotas |
| `internal/models/models.go` | Structs de domínio e page data |
| `internal/store/store.go` | Queries SQL + RunMigrations() |
| `internal/auth/auth.go` | Middlewares Protected/AdminOnly/FamilyAdminOnly/ScreenProtected |
| `internal/handler/handler.go` | Handlers HTTP + InitTemplates() + helpers |
| `internal/handler/reports.go` | Handlers de relatório PDF das telas principais |
| `internal/handler/importexport.go` | Import/export XLSX via excelize |
| `static/app.css` | Design system completo |
| `templates/base.html` | Layout + sidebar + JS utilitários |

## Produção
- Oracle Cloud Always Free ARM
- URL/IP: `http://157.151.131.79`
- SSH: `ssh -i C:/Go/finBertoldi/Oracle/ssh-key-2026-05-22.key ubuntu@157.151.131.79`
- App dir no servidor: `/home/ubuntu/finBertoldi`
- Rebuild: `docker compose -f docker-compose.prod.yml --env-file .env.prod up -d --build`
- Logs: `docker logs finbertoldi-app-1 --tail 50`
- DB: `docker exec -it finbertoldi-postgres-1 psql -U fincontrol -d fincontrol`
- Admin padrão: `admin@finbertoldi.com` / `admin123`

## Atualizar produção
```powershell
# Copiar tudo e rebuildar
scp -i C:/Go/finBertoldi/Oracle/ssh-key-2026-05-22.key -r C:/Go/finBertoldi ubuntu@157.151.131.79:/home/ubuntu/
ssh -i C:/Go/finBertoldi/Oracle/ssh-key-2026-05-22.key ubuntu@157.151.131.79 "cd /home/ubuntu/finBertoldi && docker compose -f docker-compose.prod.yml --env-file .env.prod up -d --build"

# Só CSS, sem rebuild
scp -i C:/Go/finBertoldi/Oracle/ssh-key-2026-05-22.key C:/Go/finBertoldi/static/app.css ubuntu@157.151.131.79:/home/ubuntu/finBertoldi/static/app.css
```

## Rotas principais
| Método | Rota | Handler |
|--------|------|---------|
| GET | `/` | Dashboard, requer auth |
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

## Funcionalidades
- Dashboard com receitas × despesas e histórico de 6 meses
- Despesas fixas + parcelamentos com toggle pago/mês
- Receitas recorrentes e únicas
- Investimentos por tipo + reserva de emergência
- Empréstimos com quitação parcial
- Planejamento FIRE com taxa de poupança e projeção por juros compostos
- Cadastros de categorias e cartões
- Minha Família: membros + controle de acesso por tela
- Usuários / Famílias para admin global
- Relatórios PDF em 5 telas, com botão "Relatório" em cada página
- Importar / Exportar XLSX em `/dados`, multi-aba com 5 tipos de dados
- Tema dark/light + ocultar valores

## Multi-tenancy
Todos os dados são isolados por `family_id`. Toda query que acessa dados de negócio deve incluir filtro `WHERE family_id = $X` ou equivalente, conforme a posição dos parâmetros.

## Migrações
`RunMigrations()` é idempotente e roda no startup.

Tabelas principais:
`families → users → sessions → categorias → cartoes → despesas_fixas → despesas_fixas_mes → parcelamentos → receitas → investimentos → reserva_emergencia → emprestimos → emprestimos_pagamentos → screen_permissions`

## Grupos de categoria do dashboard
- `basica` → Contas Básicas
- `cartao` → Cartão de Crédito
- `vr` → VR/VA

## Padrões de código
- Handlers seguem o formato `HandleX(db *sql.DB) http.HandlerFunc`.
- Template helpers conhecidos: `brl` (HTML span), `date`, `mesDisplay`, `sub`, `pct`, `progress`, `seq`, `json`, `contains`.
- CSS usa variáveis em `:root`.
- Use `.page-actions` para agrupar botões no page-header.
- Responsivo: tabelas viram scroll; sidebar vira overlay em `<=768px`.
- Preserve o estilo existente de `net/http`, `html/template`, SQL direto e HTMX. Não introduza frameworks novos sem necessidade explícita.

## Testes
```bash
go test ./...                          # unitários, sem banco
go test -tags=integration ./...        # integração, precisa PostgreSQL
```

Arquivos de teste conhecidos:
- `internal/handler/handler_test.go`
- `internal/store/store_test.go`
- `internal/store/integration_test.go`
- `internal/auth/auth_test.go`

## Notas para continuar o projeto com Codex
- O contexto antigo do Claude está em `CLAUDE.md` e `.claude/commands/finbertoldi.md`; use `AGENTS.md` como fonte principal daqui em diante.
- Antes de alterar queries, confira se o isolamento por `family_id` foi preservado.
- Antes de mexer em UI, leia `templates/base.html` e `static/app.css` para seguir o design system já existente.
- Para alterações que impactem produção, valide localmente quando possível e só rode comandos remotos/deploy com confirmação do usuário.
