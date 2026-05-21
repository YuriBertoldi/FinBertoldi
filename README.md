# FinBertoldi — Controle Financeiro Familiar

Sistema de finanças pessoais multi-família com dashboard, investimentos, planejamento FIRE e controle de acesso por usuário.

## Stack

| Camada | Tecnologia |
|--------|-----------|
| Backend | Go 1.22 — net/http + html/template (stdlib) |
| Frontend | HTMX 1.9.12, Chart.js 4.4.1, CSS custom (dark/glassmorphism) |
| Banco | PostgreSQL 16 |
| Auth | Sessões + bcrypt |
| Infra | Docker Compose |

---

## Rodando localmente

**Pré-requisito:** Docker + Docker Compose

```bash
docker compose up --build -d
```

Acesse: **http://localhost:8080**

| Campo | Valor padrão |
|-------|-------------|
| E-mail | `admin@finbertoldi.com` |
| Senha | `admin123` |

> Troque a senha no primeiro acesso em Usuários.

O schema do banco é criado e atualizado automaticamente no startup — não é necessário rodar SQL manualmente.

---

## Funcionalidades

- **Dashboard** — saldo do mês, gráficos de receitas × despesas (Chart.js), histórico 6 meses
- **Despesas** — fixas/recorrentes + parcelamentos de cartão com toggle pago/mês
- **Receitas** — recorrentes e únicas
- **Investimentos** — por instituição/tipo, totais por categoria, histórico
- **Reserva de Emergência** — histórico de saldo, meta de 6× e 12× despesa mensal
- **Empréstimos** — devo / emprestei, quitação total ou parcial com histórico de pagamentos
- **Planejamento FIRE** — taxa de poupança, metas FIRE (3%/3.5%/4%), projeção de juros compostos
- **Cadastros** — categorias (com cor e grupo) e cartões de crédito por família
- **Minha Família** — gerenciar membros, redefinir senhas, controle de acesso por tela por usuário
- **Usuários / Famílias** — administração global (admin only)
- **Tema dark/light** + **ocultar valores** (botão olho na sidebar)

### Controle de acesso

| Perfil | Permissões |
|--------|-----------|
| Admin global | Acesso total; gerencia todas as famílias |
| Admin de família | Gerencia membros da própria família; pode bloquear telas por usuário |
| Usuário comum | Acesso às telas liberadas pelo admin da família; pode alterar só a própria senha |

---

## Estrutura do projeto

```
finBertoldi/
├── main.go              # Wiring: App struct, rotas, startup
├── handlers.go          # Handlers HTTP (camada de apresentacao)
├── db.go                # Queries SQL + sistema de migrations (camada de dados)
├── models.go            # Structs de dominio
├── auth.go              # Sessoes, bcrypt, middlewares
├── auth_test.go         # Testes unitarios — auth
├── helpers_test.go      # Testes unitarios — helpers
├── integration_test.go  # Testes de integracao (requer PostgreSQL)
├── Dockerfile           # Multi-stage build: golang:1.22 → alpine
├── docker-compose.yml   # Ambiente de desenvolvimento
├── docker-compose.prod.yml  # Producao (Oracle Cloud)
├── .env.prod.example    # Template de variaveis de ambiente
├── migrations/
│   └── schema.sql       # Schema de referencia (documentacao)
├── scripts/
│   └── setup-oracle.sh  # Provisionamento do servidor Oracle Cloud
├── static/
│   └── app.css          # Design system completo
└── templates/
    ├── base.html        # Layout base + sidebar + JS utilitarios
    ├── login.html
    ├── dashboard.html
    ├── despesas.html
    ├── emprestimos.html
    ├── receitas.html
    ├── investimentos.html
    ├── planejamento.html
    ├── cadastros.html
    ├── minha-familia.html
    ├── usuarios.html
    └── familias.html
```

### Camadas do backend (todos `package main`)

| Arquivo | Responsabilidade |
|---------|-----------------|
| `main.go` | Inicialização, App struct, registro de rotas |
| `models.go` | Structs de domínio e page data |
| `db.go` | Queries PostgreSQL + `runMigrations()` |
| `auth.go` | Middleware `protected`, `adminOnly`, `familyAdminOnly`, `screenProtected` |
| `handlers.go` | Um handler por rota, template rendering |

---

## Migrations

O schema é versionado internamente em `db.go` (slice `migrations`). Ao subir, `runMigrations()`:

1. Cria a tabela `schema_migrations` se não existir
2. Para cada migration, verifica se já foi aplicada
3. Executa em transação as que estão pendentes
4. Registra a versão aplicada

Para adicionar uma nova alteração de schema, basta acrescentar ao slice:

```go
{
    version: 9,
    name:    "descricao_da_mudanca",
    stmts: []string{
        `ALTER TABLE tabela ADD COLUMN IF NOT EXISTS nova_coluna TYPE`,
    },
},
```

O schema de referência completo está em `migrations/schema.sql`.

---

## Variáveis de ambiente

| Variável | Padrão | Descrição |
|----------|--------|-----------|
| `DATABASE_URL` | *(montado a partir das abaixo)* | String de conexão completa (prioridade) |
| `DB_HOST` | `localhost` | Host do PostgreSQL |
| `DB_PORT` | `5432` | Porta |
| `DB_USER` | `postgres` | Usuário |
| `DB_PASS` | `postgres` | Senha |
| `DB_NAME` | `fincontrol` | Nome do banco |
| `PORT` | `8080` | Porta HTTP |

Copie `.env.prod.example` para `.env.prod` e preencha antes de subir em produção.

---

## Testes

```bash
# Unitarios (sem banco)
go test ./...

# Integracao (requer PostgreSQL rodando)
docker compose up postgres -d
go test -tags=integration ./...
```

---

## Deploy — Oracle Cloud

```bash
# Enviar arquivos
scp -i ~/Downloads/ssh-key-2026-05-19.key -r . ubuntu@141.148.34.13:/home/ubuntu/finBertoldi/

# No servidor
ssh -i ~/Downloads/ssh-key-2026-05-19.key ubuntu@141.148.34.13
cd /home/ubuntu/finBertoldi
cp .env.prod.example .env.prod  # editar com senhas reais
docker compose -f docker-compose.prod.yml --env-file .env.prod up -d --build
```

### Atualizar producao

```bash
# Windows (PowerShell)
scp -i "$env:USERPROFILE\Downloads\ssh-key-2026-05-19.key" arquivo ubuntu@141.148.34.13:/home/ubuntu/finBertoldi/

# No servidor
docker compose -f docker-compose.prod.yml --env-file .env.prod up -d --build
```
