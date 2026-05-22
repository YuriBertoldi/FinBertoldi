# FinBertoldi — Controle Financeiro Familiar

Sistema de finanças pessoais multi-família com dashboard, investimentos, planejamento FIRE, relatórios PDF e importação/exportação de dados.

## Stack

| Camada | Tecnologia |
|--------|-----------|
| Backend | Go 1.24 — net/http + html/template (stdlib) |
| Frontend | HTMX 1.9.12 · Chart.js 4.4.1 · Pico CSS v2 · CSS custom (dark/glassmorphism) |
| Banco | PostgreSQL 16 |
| Auth | Sessões + bcrypt |
| XLSX | excelize/v2 |
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

O schema é criado e atualizado automaticamente no startup — não é necessário rodar SQL manualmente.

---

## Funcionalidades

- **Dashboard** — cards: Receitas, Despesas, Sobra, **Caixa** (Receitas − pago − investido/mês), Investido/mês, Total Investido; gráficos receitas × despesas (Chart.js), histórico 6 meses
- **Despesas** — fixas/recorrentes + parcelamentos de cartão com toggle pago/mês
- **Pagamentos do Mês** — página dedicada para acompanhar o que foi pago/pendente no mês; filtros por categoria e status; abate do Caixa conforme pagamentos registrados
- **Receitas** — recorrentes e únicas; edição inline de cada registro
- **Investimentos** — por instituição/tipo, totais por categoria, histórico de aportes; edição inline de cada aporte
- **Reserva de Emergência** — funciona como **conta corrente**: lançamentos de depósito (+) e retirada (−) com data e notas; saldo = Σdepósitos − Σretiradas; depósitos contam como "investido no mês" para o Caixa; edição inline de cada lançamento
- **Empréstimos** — devo / me devem, quitação total ou parcial com histórico de pagamentos
- **Planejamento FIRE** — taxa de poupança, metas FIRE (3%/3.5%/4%), projeção de juros compostos
- **Cadastros** — categorias (com cor e grupo) e cartões de crédito por família
- **Minha Família** — gerenciar membros, redefinir senhas, controle de acesso por tela
- **Usuários / Famílias** — administração global (admin only)
- **Relatórios PDF** — botão "📄 Relatório" em cada tela; gera HTML otimizado para impressão/PDF via `Ctrl+P`
- **Importar / Exportar** — XLSX multi-aba com todos os dados; download de modelo em branco
- **Seleção e totalização** — checkboxes em todas as tabelas com barra flutuante de soma
- **Ordenação** — clique no cabeçalho de qualquer coluna para ordenar
- **Tema dark/light** + **ocultar valores** (botão olho na sidebar)

### Controle de acesso

| Perfil | Permissões |
|--------|-----------|
| Admin global | Acesso total; gerencia todas as famílias |
| Admin de família | Gerencia membros da própria família; bloqueia telas por usuário |
| Usuário comum | Acesso às telas liberadas; altera só a própria senha |

---

## Estrutura do projeto

```
finBertoldi/
├── main.go                          # Wiring: todas as rotas + startup
├── go.mod / go.sum
├── Dockerfile                       # Multi-stage: golang:1.24 → alpine
├── docker-compose.yml               # Desenvolvimento local
├── docker-compose.prod.yml          # Produção (Oracle Cloud)
├── .env.prod.example                # Template de variáveis de ambiente
├── internal/
│   ├── models/
│   │   └── models.go                # Structs de domínio e page data
│   ├── store/
│   │   ├── store.go                 # Queries PostgreSQL + RunMigrations()
│   │   ├── store_test.go            # Testes unitários
│   │   └── integration_test.go      # Testes de integração (build tag: integration)
│   ├── auth/
│   │   ├── auth.go                  # Middlewares + sessões + bcrypt
│   │   └── auth_test.go             # Testes unitários
│   └── handler/
│       ├── handler.go               # Handlers HTTP + InitTemplates() + helpers
│       ├── reports.go               # Handlers relatórios PDF (5 telas)
│       ├── importexport.go          # Import/Export XLSX
│       └── handler_test.go          # Testes unitários
├── static/
│   └── app.css                      # Design system completo
└── templates/
    ├── base.html                    # Layout base + sidebar + JS utilitários (initTableSort, initSelecao, initTableFilter)
    ├── login.html
    ├── dashboard.html
    ├── despesas.html
    ├── pagamentos.html              # Pagamentos do Mês (página dedicada)
    ├── receitas.html
    ├── investimentos.html
    ├── emprestimos.html
    ├── planejamento.html
    ├── cadastros.html
    ├── minha-familia.html
    ├── usuarios.html
    ├── familias.html
    ├── importexport.html            # Importar / Exportar XLSX
    ├── relatorio-base.html          # Layout base dos relatórios PDF
    ├── relatorio-dashboard.html
    ├── relatorio-despesas.html
    ├── relatorio-receitas.html
    ├── relatorio-investimentos.html
    └── relatorio-emprestimos.html
```

---

## Importar / Exportar

Acesse `/dados` (sidebar: **📊 Importar / Exportar**).

| Ação | Descrição |
|------|-----------|
| **Exportar meus dados** | Baixa XLSX com todos os dados atuais em 5 abas |
| **Baixar modelo em branco** | XLSX com cabeçalhos + linha de exemplo para preencher |
| **Importar** | Faz upload de XLSX no mesmo formato; **adiciona** aos dados existentes |

**Formato das abas:**

| Aba | Colunas |
|-----|---------|
| Despesas Fixas | Nome · Valor · Categoria · Ativa (S/N) |
| Parcelamentos | Descrição · Cartão · Valor Parcela · Parcela Atual · Total Parcelas · Data Início |
| Receitas | Descrição · Valor · Data · Tipo · Recorrente (S/N) |
| Investimentos | Instituição · Tipo · Valor · Data · Notas |
| Empréstimos | Pessoa · Valor · Direção (devo/me_devem) · Data · Notas |

---

## Relatórios PDF

Cada tela principal tem um botão **📄 Relatório** que abre uma página otimizada para impressão. Use `Ctrl+P` → **Salvar como PDF** no navegador.

Rotas disponíveis:
- `GET /relatorio/dashboard?mes=YYYY-MM`
- `GET /relatorio/despesas`
- `GET /relatorio/receitas`
- `GET /relatorio/investimentos`
- `GET /relatorio/emprestimos`

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

---

## Testes

```bash
# Unitários (sem banco, <1s)
go test ./...

# Integração (requer PostgreSQL rodando)
docker compose up postgres -d
go test -tags=integration ./...
```

Ver `TESTES.md` para detalhes completos.

---

## Deploy — Oracle Cloud

**Servidor:** `157.151.131.79` (VM.Standard.E2.1.Micro — Always Free)

```powershell
# Enviar e rebuildar (PowerShell)
scp -i C:/Go/finBertoldi/Oracle/ssh-key-2026-05-22.key -r C:/Go/finBertoldi ubuntu@157.151.131.79:/home/ubuntu/
ssh -i C:/Go/finBertoldi/Oracle/ssh-key-2026-05-22.key ubuntu@157.151.131.79 "cd /home/ubuntu/finBertoldi && docker compose -f docker-compose.prod.yml --env-file .env.prod up -d --build"
```

```bash
# Logs
docker logs finbertoldi-app-1 --tail 50 -f

# Acessar banco
docker exec -it finbertoldi-postgres-1 psql -U fincontrol -d fincontrol
```
