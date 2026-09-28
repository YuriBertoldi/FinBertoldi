# Modulo: integrations

**Pacote:** `internal/integrations`
**Arquivos:** `scheduler.go`, `bcb.go`, `cotacoes.go`, `telegram.go`, `brasilapi.go`, `sheets.go`

## Responsabilidade

Integracoes gratuitas opcionais que enriquecem o controle financeiro. Cada integracao pode ser ativada/desativada por familia via UI (`/integracoes`). Dados globais (BCB, cotacoes, feriados) sao compartilhados entre familias; dados por familia (Telegram, Sheets) usam config KV especifica.

## Scheduler (scheduler.go)

Goroutine iniciada no startup via `StartScheduler(db)`:

| Ticker | Intervalo | Funcao | Escopo |
|--------|-----------|--------|--------|
| globalTicker | 6h | `runGlobalIntegrations` | BCB + Cotacoes + Feriados |
| familyTicker | 24h | `runFamilyIntegrations` | Telegram (por familia) |

Executa imediatamente na primeira vez (10s apos startup).

## Integracoes

### BCB — Banco Central do Brasil (bcb.go)

Coleta Selic, CDI e IPCA via API publica do BCB.

| Indicador | Serie BCB | Descricao |
|-----------|-----------|-----------|
| Selic | 432 | Taxa Selic meta |
| CDI | 4389 | Taxa CDI |
| IPCA | 433 | Inflacao mensal |

**API:** `https://api.bcb.gov.br/dados/serie/bcdata.sgs.{serie}/dados/ultimos/1?formato=json`

### Cotacoes — AwesomeAPI (cotacoes.go)

Coleta cotacoes de moedas em tempo real.

| Moeda | Par | Tipo |
|-------|-----|------|
| USD | USD-BRL | Dolar americano |
| EUR | EUR-BRL | Euro |
| BTC | BTC-BRL | Bitcoin |

**API:** `https://economia.awesomeapi.com.br/last/USD-BRL,EUR-BRL,BTC-BRL`

Trata erro 429 (rate limit) graciosamente.

### Telegram Bot (telegram.go)

Envia alertas e resumos via Telegram. Requer config por familia:

| Campo | Tipo | Descricao |
|-------|------|-----------|
| bot_token | password | Token do bot (@BotFather) |
| chat_id | text | ID do chat/grupo |

**Funcionalidades:**
- `SendAlertaVencimento` — alerta de despesas pendentes no mes
- `SendResumoSemanal` — resumo com entradas/saidas/saldo
- `TestTelegram` — mensagem de teste

### BrasilAPI — Feriados (brasilapi.go)

Importa feriados nacionais do ano corrente.

**API:** `https://brasilapi.com.br/api/feriados/v1/{ano}`

Os feriados aparecem no widget de indicadores do dashboard.

### Google Sheets (sheets.go)

Exportacao de dados para Google Sheets (stub — autenticacao JWT nao implementada).

## Configuracao via UI

A pagina `/integracoes` exibe cards para cada integracao com:
- Toggle ativa/inativa
- Campos de configuracao (KV)
- Botao "Testar" (quando ativa)
- Botao "Salvar"

Configs salvas na tabela `integracoes_kv` com chave composta `(family_id, integracao, chave)`.

## Dashboard Widget

Quando ha dados coletados, o dashboard exibe o widget "Indicadores" na coluna direita:
- Selic, CDI, IPCA (se BCB ativo)
- USD, EUR, BTC (se cotacoes ativo)
- Proximos feriados (se BrasilAPI ativo)
