# FinBertoldi — Controle Financeiro Familiar

Sistema de financas pessoais multi-familia com dashboard, investimentos, planejamento FIRE, relatorios PDF, importacao/exportacao de dados e integracao bancaria automatica.

## Stack

| Camada | Tecnologia |
|--------|-----------|
| Backend | Go 1.24 — net/http + html/template (stdlib) |
| Frontend | HTMX 1.9.12 · Chart.js 4.4.1 · Pico CSS v2 · CSS custom (dark/glassmorphism) |
| Banco | PostgreSQL 16 |
| Auth | Sessoes + bcrypt |
| XLSX | excelize/v2 |
| OFX | ofxgo (parser de extratos bancarios) |
| Integracao | Pluggy API (microservico separado) |
| Infra | Docker Compose (3 containers) |

---

## Rodando localmente

**Pre-requisito:** Docker + Docker Compose

```bash
docker compose up --build -d
```

Acesse: **http://localhost:8080**

O schema e criado e atualizado automaticamente no startup (12 migrations idempotentes).

> Credenciais de acesso padrao estao em `PRODUCAO.md` (ignorado pelo git). Crie o seu proprio `.env.prod` a partir de `.env.prod.example`.

---

## Arquitetura

```
                    ┌──────────────┐
                    │   Browser    │
                    │  (HTMX)     │
                    └──────┬───────┘
                           │ :8080
                    ┌──────┴───────┐
                    │   App (Go)   │
                    │   main.go    │
                    └──┬───────┬───┘
                       │       │ proxy HTTP
                ┌──────┴──┐ ┌──┴──────────────┐
                │ Postgres │ │ Pluggy Service  │
                │  :5432   │ │   :8081 (Go)    │
                └──────────┘ └────────┬────────┘
                                      │ HTTPS
                               ┌──────┴───────┐
                               │  Pluggy API   │
                               │ api.pluggy.ai │
                               └──────────────┘
```

**3 containers Docker:**
1. **app** — aplicacao principal Go (porta 8080)
2. **postgres** — PostgreSQL 16 Alpine
3. **pluggy-service** — microservico de integracao bancaria (porta 8081)

---

## Funcionalidades

- **Dashboard** — cards: Receitas, Despesas, Sobra, Caixa, Investido/mes, Total Investido; graficos (Chart.js), historico 6 meses
- **Despesas** — fixas/recorrentes + parcelamentos de cartao com toggle pago/mes
- **Pagamentos do Mes** — pagina dedicada com filtros por categoria e status
- **Receitas** — recorrentes e unicas; edicao inline
- **Investimentos** — por instituicao/tipo, totais por categoria, edicao inline
- **Reserva de Emergencia** — depositos e retiradas com saldo
- **Emprestimos** — devo / me devem, quitacao total ou parcial
- **Planejamento FIRE** — taxa de poupanca, metas FIRE, projecao de juros compostos
- **Transacoes Bancarias** — importacao de extratos CSV e OFX; conversao para despesa/receita; deduplicacao por FitID
- **Integracao Pluggy** — conexao automatica com bancos via widget; sync periodico (admin only)
- **Cadastros** — categorias, cartoes, configuracao de integracao bancaria
- **Minha Familia** — membros, senhas, controle de acesso por tela
- **Usuarios / Familias** — administracao global (admin only)
- **Relatorios PDF** — HTML otimizado para impressao via `Ctrl+P`
- **Importar / Exportar** — XLSX multi-aba com todos os dados
- **Selecao e totalizacao** — checkboxes com barra flutuante de soma
- **Ordenacao** — clique no cabecalho de qualquer coluna
- **Tema dark/light** + **ocultar valores**

### Controle de acesso

| Perfil | Permissoes |
|--------|-----------|
| Admin global | Acesso total; gerencia familias; configura integracoes |
| Admin de familia | Gerencia membros; bloqueia telas por usuario |
| Usuario comum | Acesso as telas liberadas |

---

## Estrutura do projeto

```
finBertoldi/
├── main.go                          # Wiring: rotas + startup
├── Dockerfile                       # Multi-stage: golang:1.24 → alpine
├── docker-compose.yml               # Desenvolvimento local
├── docker-compose.prod.yml          # Producao
├── .env.prod.example                # Template de variaveis de ambiente
├── internal/
│   ├── models/models.go             # Structs de dominio e page data
│   ├── store/store.go               # Queries PostgreSQL + RunMigrations() (v1-v12)
│   ├── auth/auth.go                 # Middlewares + sessoes + bcrypt
│   └── handler/
│       ├── handler.go               # Handlers HTTP + Pluggy admin
│       ├── bankimport.go            # Parsers CSV/OFX + handlers transacoes
│       ├── reports.go               # Relatorios PDF (5 telas)
│       └── importexport.go          # Import/Export XLSX
├── static/app.css                   # Design system
├── templates/                       # Templates HTML (18 paginas)
├── pluggy-service/                  # Microservico de integracao bancaria
│   ├── main.go                      # HTTP server + sync scheduler
│   ├── pluggy/client.go             # REST client Pluggy API
│   ├── pluggy/models.go             # Structs API Pluggy
│   └── sync/sync.go                 # Logica de sincronizacao
└── documentacao/                    # Documentacao detalhada por modulo
    ├── 01-visao-geral.md
    ├── 02-models.md
    ├── 03-store.md
    ├── 04-auth.md
    ├── 05-handler.md
    ├── 06-pluggy-service.md
    ├── 07-templates.md
    ├── 08-rotas.md
    ├── 09-banco-de-dados.md
    ├── 10-deploy.md
    └── swagger.yaml                 # Documentacao OpenAPI 3.0
```

---

## Integracao Bancaria

### Importacao Manual (CSV/OFX)

Na tela **Transacoes** (`/transacoes`):
- **CSV**: upload com auto-deteccao de separador e formato BR
- **OFX**: upload de arquivo OFX/QFX
- Deduplicacao automatica por FitID
- Conversao para despesa ou receita com um clique

### Integracao Automatica (Pluggy)

Configuracao em **Cadastros** > tab **Integracao Bancaria** (admin only):
1. Preencher credenciais Pluggy
2. Ativar integracao e salvar
3. Conectar banco via Pluggy Connect Widget
4. Transacoes sincronizam automaticamente a cada 6h

---

## Variaveis de ambiente

### App principal

| Variavel | Padrao | Descricao |
|----------|--------|-----------|
| `DATABASE_URL` | — | String de conexao completa (prioridade) |
| `DB_HOST` | `localhost` | Host do PostgreSQL |
| `DB_PORT` | `5432` | Porta |
| `DB_USER` | `postgres` | Usuario |
| `DB_PASS` | `postgres` | Senha |
| `DB_NAME` | `fincontrol` | Nome do banco |
| `PORT` | `8080` | Porta HTTP |
| `SESSION_SECRET` | — | Chave para cookies de sessao |
| `TZ` | `America/Sao_Paulo` | Timezone |

### Pluggy Service

| Variavel | Padrao | Descricao |
|----------|--------|-----------|
| `DATABASE_URL` | — | String de conexao PostgreSQL |
| `PLUGGY_CLIENT_ID` | — | Client ID (opcional, le do banco) |
| `PLUGGY_CLIENT_SECRET` | — | Client Secret (opcional, le do banco) |
| `SYNC_INTERVAL` | `6h` | Intervalo de sync automatico |
| `PORT` | `8081` | Porta HTTP |

---

## Testes

```bash
# Unitarios (sem banco, <1s)
go test ./...

# Integracao (requer PostgreSQL rodando)
docker compose up postgres -d
go test -tags=integration ./...
```

---

## Documentacao

Documentacao detalhada por modulo em `documentacao/`.
API documentada em formato OpenAPI 3.0 em `documentacao/swagger.yaml`.

Dados de acesso e deploy em `PRODUCAO.md` (ignorado pelo git).
