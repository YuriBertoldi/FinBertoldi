# FinBertoldi

**Sistema completo de controle financeiro familiar** — multi-tenant, com dashboard interativo, planejamento FIRE, integracao bancaria automatica e indicadores economicos em tempo real.

![Go](https://img.shields.io/badge/Go-1.27-00ADD8?logo=go&logoColor=white)
![PostgreSQL](https://img.shields.io/badge/PostgreSQL-16-4169E1?logo=postgresql&logoColor=white)
![HTMX](https://img.shields.io/badge/HTMX-1.9-3366CC)
![Docker](https://img.shields.io/badge/Docker-Compose-2496ED?logo=docker&logoColor=white)
![License](https://img.shields.io/badge/license-MIT-green)

---

## Destaques

- **Multi-tenant** — cada familia tem dados isolados; controle de acesso granular por tela
- **Zero JavaScript frameworks** — SSR com Go templates + HTMX para interatividade
- **Dashboard rico** — cards de resumo, 3 graficos (Chart.js), toggle pago/pendente em tempo real
- **Planejamento FIRE** — taxa de poupanca, reserva de emergencia, projecao de independencia financeira
- **Integracao bancaria** — importacao CSV/OFX + sync automatico via Pluggy API
- **Indicadores economicos** — Selic, CDI, IPCA, USD, EUR, BTC (atualizados automaticamente)
- **Relatorios PDF** — 5 relatorios otimizados para impressao
- **Contas do mes** — resumo pronto para WhatsApp, envio por e-mail e Telegram, e envio mensal automatico no dia escolhido
- **Google Sheets** — planilha atualizada diariamente com contas, receitas, parcelamentos e investimentos
- **Dashboard personalizavel** — cada usuario escolhe quais widgets aparecem, a ordem e a coluna
- **Mobile first** — listas viram cartoes, menu proprio e indicador de carregamento
- **Tema dark/light** com glassmorphism + opcao de ocultar valores

---

## Screenshots

| Dashboard | Despesas |
|:---------:|:--------:|
| ![Dashboard](docs/screenshots/dashboard.png) | ![Despesas](docs/screenshots/despesas.png) |

| Planejamento FIRE | Investimentos |
|:-----------------:|:-------------:|
| ![Planejamento](docs/screenshots/planejamento.png) | ![Investimentos](docs/screenshots/investimentos.png) |

*Dados ficticios — conta demo para demonstracao*

---

## Arquitetura

```
Browser (HTMX) ──── :443 ────► Caddy (HTTPS)
                                    │
                                    ▼ :8080
                            ┌───────────────┐
                            │   App (Go)    │
                            │   main.go     │
                            └──┬─────────┬──┘
                               │         │ proxy HTTP
                        ┌──────┴──┐  ┌───┴──────────────┐
                        │Postgres │  │ Pluggy Service   │
                        │  :5432  │  │   :8081 (Go)     │
                        └─────────┘  └────────┬─────────┘
                                              │ HTTPS
                                       ┌──────┴────────┐
                                       │  Pluggy API   │
                                       │ api.pluggy.ai │
                                       └───────────────┘
```

**3 containers Docker** + Caddy como reverse proxy HTTPS em producao.

---

## Stack tecnica

| Camada | Tecnologia |
|--------|-----------|
| Backend | Go 1.27 — `net/http` + `html/template` (stdlib puro, sem frameworks) |
| Frontend | HTMX 1.9.12 · Chart.js 4.4.1 · Pico CSS v2 · CSS custom (dark/glassmorphism) |
| Banco de dados | PostgreSQL 16 · lib/pq · 23 migrations automaticas |
| Autenticacao | Sessoes HTTP-only + bcrypt · recuperacao de senha via email · troca de senha encerra as outras sessoes |
| Importacao | excelize/v2 (XLSX) · ofxgo (OFX/QFX) · CSV com auto-deteccao |
| Integracoes | Pluggy API · BCB · AwesomeAPI · BrasilAPI · Telegram Bot · Google Sheets · SMTP (Gmail) |
| Infra | Docker Compose · Caddy (HTTPS/Let's Encrypt) · Oracle Cloud |

---

## Funcionalidades

### Financeiro
- **Dashboard** — cards de resumo (Receitas, Despesas, Sobra, Caixa, Patrimonio), graficos de evolucao, historico 6 meses
- **Despesas fixas** — recorrentes com toggle pago/pendente por mes
- **Alerta de orcamento** — aviso por e-mail e/ou Telegram (opcao em Notificacoes) quando uma categoria chega a 80% e quando passa de 100% do orcamento no mes, uma vez por faixa
- **Orcamento por categoria** — limite mensal opcional por categoria; widget "Orcamento do mes" soma despesas fixas e debitos do extrato (sem contar duas vezes transacoes ja convertidas) com barra verde/amarela (80%)/vermelha (acima de 100%)
- **Parcelamentos** — cartao de credito com progresso visual + suporte a financiamentos com antecipacao e desconto racional
- **Receitas** — recorrentes e unicas, edicao inline
- **Investimentos** — por instituicao/tipo, totais por categoria
- **Reserva de emergencia** — depositos e retiradas com saldo
- **Emprestimos** — devo/me devem, quitacao total ou parcial
- **Fluxo de caixa projetado** — proximos 12 meses (receitas, despesas fixas e parcelas restantes, sem as antecipadas) em widget com grafico no dashboard e tabela no relatorio geral; avisa o primeiro mes em que as despesas passam as receitas
- **Metas financeiras** — objetivo com valor-alvo e prazo (no Planejamento); aportes e retiradas registrados na meta, progresso, quanto falta e quanto guardar por mes ate o prazo
- **Planejamento FIRE** — taxa de poupanca, metas FIRE (conservador/moderado/agressivo), reserva ideal 6-12 meses

### Bancario
- **Importacao CSV/OFX** — auto-deteccao de formato, deduplicacao por FitID
- **Integracao Pluggy** — conexao automatica com bancos brasileiros, sync a cada 6h
- **Regras de categorizacao** — "categoria do banco e X" ou "descricao contem X" -> categoria da familia (Cadastros -> Regras); aplicadas em transacoes pendentes ao criar a regra, a cada importacao e de hora em hora. Sem elas, o extrato do Pluggy (categorias em ingles) nao entra no orcamento
- **Conciliacao** — vincular transacoes a despesas/receitas existentes, auto-match por valor/data

### Integracoes gratuitas (ativaveis por familia)
- **BCB** — Selic, CDI, IPCA (API Banco Central)
- **AwesomeAPI** — cotacoes USD, EUR, BTC
- **BrasilAPI** — feriados nacionais
- **Telegram Bot** — contas do mes e alertas de despesas pendentes
- **Google Sheets** — abas Resumo, Contas do mes, Receitas, Parcelamentos, Investimentos e Transacoes (Service Account, 1x por dia)
- Widget de indicadores no dashboard
- Cada integracao tem um botao **!** explicando para que serve e como configurar

### Contas do mes e notificacoes
- **Pagamentos → 📱 WhatsApp** — previa da mensagem, copiar, abrir no WhatsApp, enviar por Telegram ou e-mail
- **🔔 Notificacoes** (por usuario) — envio mensal automatico por e-mail e/ou Telegram, no dia escolhido (1–31), com as contas do mes atual ou do mes seguinte

### Administrativo
- **Multi-familia** — dados isolados por familia
- **Controle de acesso** — admin global, admin de familia, usuario com telas bloqueaveis
- **Import/Export XLSX** — todas as entidades em planilha multi-aba
- **Relatorios PDF** — dashboard, despesas, receitas, investimentos, emprestimos
- **Recuperacao de senha** — email com token de reset (SMTP configuravel)

### UX
- Tema dark/light com persistencia em localStorage
- Ocultar valores monetarios (privacidade)
- Ordenacao por coluna em todas as tabelas
- Selecao com barra flutuante de totalizacao
- Filtros por busca textual e selects
- Sidebar responsiva com menu mobile e barra superior propria
- Tabelas viram cartoes no celular (selecao, acoes laterais)
- Tela de carregamento ao navegar/salvar e botoes com indicador de ocupado
- Escala tipografica unica (8 tamanhos)

---

## Controle de acesso

| Perfil | Permissoes |
|--------|-----------|
| **Admin global** | Acesso total · gerencia familias e usuarios · configura integracoes |
| **Admin de familia** | Gerencia membros · bloqueia/libera telas por usuario |
| **Usuario comum** | Acesso apenas as telas liberadas pelo admin da familia |

---

## Rodando localmente

**Pre-requisito:** Docker + Docker Compose

```bash
# Clonar e subir
git clone https://github.com/YuriBertoldi/FinBertoldi.git
cd FinBertoldi
docker compose up --build -d
```

Acesse: **http://localhost:8080**

No primeiro startup, um usuario admin e criado automaticamente com senha aleatoria exibida nos logs:

```bash
docker compose logs app | grep "admin criado"
```

> As 23 migrations do banco sao executadas automaticamente.

---

## Variaveis de ambiente

### App principal

| Variavel | Padrao | Descricao |
|----------|--------|-----------|
| `DATABASE_URL` | — | Connection string completa (prioridade) |
| `DB_HOST` | `localhost` | Host PostgreSQL |
| `DB_PORT` | `5432` | Porta |
| `DB_USER` | `fincontrol` | Usuario |
| `DB_PASS` | `fincontrol` | Senha |
| `DB_NAME` | `fincontrol` | Nome do banco |
| `PORT` | `8080` | Porta HTTP |
| `ADMIN_EMAIL` | `admin@localhost` | Email do admin inicial |
| `ADMIN_PASSWORD` | *(aleatorio)* | Senha do admin inicial |
| `SMTP_USER` | — | Conta que envia os e-mails (reset de senha e contas do mes) |
| `SMTP_PASS` | — | Senha (para Gmail: senha de app) |
| `SMTP_HOST` | `smtp.gmail.com` se `SMTP_USER` for Gmail | Servidor SMTP |
| `SMTP_PORT` | `587` | Porta (`465` = TLS direto) |
| `SMTP_FROM` | `FinBertoldi <SMTP_USER>` | Remetente |
| `TZ` | `America/Sao_Paulo` | Timezone |

### Pluggy Service

| Variavel | Padrao | Descricao |
|----------|--------|-----------|
| `DATABASE_URL` | — | Connection string PostgreSQL |
| `PLUGGY_CLIENT_ID` | — | Client ID Pluggy (ou via UI) |
| `PLUGGY_CLIENT_SECRET` | — | Client Secret Pluggy (ou via UI) |
| `WEBHOOK_TOKEN` | — | Token de validacao de webhooks |
| `SYNC_INTERVAL` | `6h` | Intervalo de sync automatico |
| `PORT` | `8081` | Porta HTTP |

---

## Estrutura do projeto

```
finBertoldi/
├── main.go                          # Wiring: rotas + startup
├── Dockerfile                       # Multi-stage: golang:1.27 → alpine
├── docker-compose.yml               # Desenvolvimento local
├── docker-compose.prod.yml          # Producao
├── internal/
│   ├── models/models.go             # Structs de dominio e page data
│   ├── store/                       # Queries SQL, um arquivo por dominio + migrations.go (v1-v23)
│   ├── auth/
│   │   ├── auth.go                  # Middlewares + sessoes + bcrypt
│   │   └── email.go                 # Envio de e-mail SMTP (Gmail embutido)
│   ├── handler/
│   │   ├── <tela>.go                # Um arquivo por tela (despesas.go, receitas.go, ...)
│   │   ├── autenticacao.go          # Login, cadastro, recuperacao de senha
│   │   ├── templates.go             # InitTemplates + render
│   │   ├── bankimport.go            # Parsers CSV/OFX + conciliacao
│   │   ├── reports.go               # Relatorios PDF (5 telas)
│   │   ├── importexport.go          # Import/Export XLSX
│   │   └── whatsapp.go              # Contas do mes + Notificacoes
│   ├── contas/                      # Resumo das contas do mes + envio e-mail/Telegram + agendador
│   └── integrations/
│       ├── scheduler.go             # Scheduler periodico (goroutines)
│       ├── bcb.go                   # API Banco Central (Selic/CDI/IPCA)
│       ├── cotacoes.go              # AwesomeAPI (USD/EUR/BTC)
│       ├── telegram.go              # Telegram Bot (contas do mes, alertas)
│       ├── brasilapi.go             # BrasilAPI (feriados)
│       └── sheets.go                # Google Sheets (Service Account, 6 abas)
├── templates/                       # 24 templates HTML
├── static/app.css                   # Design system completo
├── pluggy-service/                  # Microservico integracao bancaria
│   ├── main.go                      # HTTP server + sync scheduler
│   ├── pluggy/client.go             # REST client Pluggy API
│   └── sync/sync.go                 # Logica de sincronizacao
└── documentacao/                    # Documentacao detalhada (11 arquivos)
```

---

## Testes

```bash
# Unitarios (sem banco)
go test ./...

# Integracao (requer PostgreSQL rodando)
docker compose up postgres -d
go test -tags=integration ./...
```

O GitHub Actions roda os dois a cada PR e a cada push na `main` (com um PostgreSQL de teste). Detalhes em [`TESTES.md`](TESTES.md).

---

## Deploy em producao

O sistema roda em Oracle Cloud com Caddy como reverse proxy HTTPS (Let's Encrypt automatico). O merge na `main` so roda os testes; a publicacao e manual, pelo botao *Run workflow* na `main`, sempre com backup do banco antes. As migrations rodam no startup do app. Detalhes em [`documentacao/10-deploy.md`](documentacao/10-deploy.md).

```bash
# Copiar .env.prod.example e configurar
cp .env.prod.example .env.prod

# Subir em producao
docker compose -f docker-compose.prod.yml --env-file .env.prod up -d --build
```

Detalhes completos em [`documentacao/10-deploy.md`](documentacao/10-deploy.md).

---

## Documentacao

Documentacao detalhada por modulo em [`documentacao/`](documentacao/):

| Arquivo | Conteudo |
|---------|----------|
| [01-visao-geral](documentacao/01-visao-geral.md) | Arquitetura, stack, multi-tenancy |
| [02-models](documentacao/02-models.md) | Structs de dominio e page data |
| [03-store](documentacao/03-store.md) | Queries SQL, migrations, funcoes por dominio |
| [04-auth](documentacao/04-auth.md) | Autenticacao, middlewares, recuperacao de senha |
| [05-handler](documentacao/05-handler.md) | Handlers HTTP, template helpers |
| [06-pluggy-service](documentacao/06-pluggy-service.md) | Microservico de integracao bancaria |
| [07-templates](documentacao/07-templates.md) | Templates HTML, layout, componentes |
| [08-rotas](documentacao/08-rotas.md) | Mapa completo de rotas |
| [09-banco-de-dados](documentacao/09-banco-de-dados.md) | Schema de todas as tabelas |
| [10-deploy](documentacao/10-deploy.md) | Deploy Oracle Cloud + Caddy |
| [11-integracoes](documentacao/11-integracoes.md) | BCB, cotacoes, Telegram, BrasilAPI, Google Sheets, contas do mes e notificacoes |

---

## Licenca

MIT
