# Skill: Contexto FinBertoldi

Você está trabalhando no **FinBertoldi** — sistema de controle financeiro familiar.

## Projeto
- **Stack:** Go 1.22 · net/http · html/template · PostgreSQL 16 · HTMX 1.9.12 · Chart.js · CSS puro
- **Usuário:** Yuri Bertoldi (`ybulhoesbertoldi@gmail.com`)
- **Local:** `C:/Go/fincontrol`

## Produção (Oracle Cloud Always Free)
- **IP:** `141.148.34.13` → http://141.148.34.13
- **SSH:** `ssh -i "$env:USERPROFILE\Downloads\ssh-key-2026-05-19.key" ubuntu@141.148.34.13`
- **App dir:** `/home/ubuntu/fincontrol`
- **Rebuild:** `docker compose -f docker-compose.prod.yml --env-file .env.prod up -d --build`
- **Logs:** `docker logs fincontrol-app-1 --tail 50`
- **DB:** `docker exec -it fincontrol-postgres-1 psql -U fincontrol -d fincontrol`
- **Credenciais:** `admin@finbertoldi.com` / `admin123`

## Enviar para produção
```powershell
# Arquivos Go/templates (requer rebuild)
scp -i "$env:USERPROFILE\Downloads\ssh-key-2026-05-19.key" "C:\Go\fincontrol\ARQUIVO" ubuntu@141.148.34.13:/home/ubuntu/fincontrol/ARQUIVO

# Só CSS (sem rebuild)
scp -i "$env:USERPROFILE\Downloads\ssh-key-2026-05-19.key" "C:\Go\fincontrol\static\app.css" ubuntu@141.148.34.13:/home/ubuntu/fincontrol/static/app.css
```

## Arquivos principais
| Arquivo | Responsabilidade |
|---------|-----------------|
| `main.go` | Rotas HTTP |
| `handlers.go` | Handlers HTTP |
| `db.go` | Queries + migração automática no startup |
| `models.go` | Structs de domínio |
| `auth.go` | Middleware `protected`/`adminOnly`, cookies |
| `helpers.go` | `brl()`, `parseBRL()`, funções de data |
| `static/app.css` | Design system completo |
| `templates/base.html` | Layout + sidebar toggle mobile |
| `templates/cadastros.html` | CRUD categorias e cartões |
| `templates/despesas.html` | Tabs: Fixas / Parcelamentos / Empréstimos |

## Multi-tenancy
Todos os dados isolados por `family_id`. Toda query inclui `WHERE family_id = $X`.

## Migração (`dbAutoMigrate`)
Roda no startup. Ordem de criação:
`families → users → sessions → categorias → cartoes → despesas_fixas → despesas_fixas_mes → parcelamentos → receitas → investimentos → reserva_emergencia → emprestimos`

## Grupos de categoria (dashboard)
- `basica` → Contas Básicas
- `cartao` → Cartão de Crédito  
- `vr` → VR/VA

## Padrões de código
- Handlers: `handleX(app *App) http.HandlerFunc`
- Template helpers: `brl`, `date`, `progress`
- CSS: variáveis em `:root`; `.badge-grupo-{basica|cartao|vr}`
- Responsivo: tabelas → cards em ≤640px; sidebar → overlay em ≤768px
- `font-size: 16px` em inputs (evita zoom iOS)

## Pendências
- [ ] HTTPS (nginx + Let's Encrypt — necessita domínio)
- [ ] Tela de registro público
