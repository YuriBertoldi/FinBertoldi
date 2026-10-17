# Modulo: integrations e contas

**Pacotes:**
- `internal/integrations` — `scheduler.go`, `bcb.go`, `cotacoes.go`, `telegram.go`, `brasilapi.go`, `sheets.go`
- `internal/contas` — `texto.go`, `envio.go` (resumo das contas do mes + envio por e-mail/Telegram)

## Responsabilidade

Integracoes gratuitas opcionais que enriquecem o controle financeiro. Cada integracao pode ser ativada/desativada por familia via UI (`/integracoes`). Dados globais (BCB, cotacoes, feriados) sao compartilhados entre familias; dados por familia (Telegram, Sheets) usam config KV especifica (`integracoes_kv`).

## Agendadores

### `integrations.StartScheduler(db)` (scheduler.go)

Goroutine iniciada no startup:

| Ticker | Intervalo | Funcao | Escopo |
|--------|-----------|--------|--------|
| globalTicker | 6h | `runGlobalIntegrations` | BCB + Cotacoes + Feriados |
| familyTicker | 24h | `runFamilyIntegrations` | Google Sheets (familias com `sheets` ativa) + alerta Telegram |

Executa imediatamente na primeira vez (10s apos startup).

### `contas.StartScheduler(db)` (contas/envio.go)

Verifica **de hora em hora** quem deve receber o envio mensal das contas (ver [Contas do mes](#contas-do-mes-internalcontas)).

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

| Moeda | Par | Tipo |
|-------|-----|------|
| USD | USD-BRL | Dolar americano |
| EUR | EUR-BRL | Euro |
| BTC | BTC-BRL | Bitcoin |

**API:** `https://economia.awesomeapi.com.br/last/USD-BRL,EUR-BRL,BTC-BRL`

Trata erro 429 (rate limit) graciosamente — a proxima execucao tenta de novo.

### Telegram Bot (telegram.go)

Config por familia (admin, em `/integracoes`):

| Campo | Tipo | Descricao |
|-------|------|-----------|
| bot_token | segredo | Token do bot (@BotFather), formato `123456789:AA...` |
| chat_id | text | Chat padrao da familia (cada usuario pode informar o seu em `/notificacoes`) |

**Funcoes:**
- `SendTelegramMessage(token, chatID, html)` — envio com `parse_mode=HTML`; erros da API sao traduzidos por `explicarErroTelegram` (token invalido/revogado, chat nao encontrado, bot bloqueado, falta de `/start`)
- `ValidarTokenTelegram(token)` — valida o formato antes de salvar
- `TelegramBotUsername(token)` — `getMe` com cache em memoria; usado para mostrar o `@bot` em Notificacoes
- `SendAlertaVencimento` — alerta de despesas pendentes no mes
- `TestTelegram` — mensagem de teste (ao salvar/ativar e no botao Testar)

> O usuario precisa enviar `/start` ao bot antes: o Telegram nao deixa o bot iniciar a conversa.

### BrasilAPI — Feriados (brasilapi.go)

Importa feriados nacionais do ano corrente. **API:** `https://brasilapi.com.br/api/feriados/v1/{ano}`

### Google Sheets (sheets.go)

Mantem uma planilha Google atualizada com os dados da familia.

**Configuracao (por familia):**

| Campo | Tipo | Descricao |
|-------|------|-----------|
| spreadsheet_id | text | Link ou ID da planilha (`ExtrairIDPlanilha` aceita o link completo) |
| service_account_json | segredo | Conteudo do `.json` da chave da Service Account (Google Cloud) |
| ultima_exportacao | interno | Data/hora da ultima exportacao bem-sucedida |

A planilha deve ser compartilhada como **Editor** com o `client_email` da Service Account (exibido na tela apos salvar) e a **Google Sheets API** deve estar ativa no projeto.

**Fluxo de `ExportToSheet(db, fid)`:**
1. `ParseServiceAccount` valida o JSON e carrega a chave RSA (PKCS#8 ou PKCS#1)
2. `TokenAcesso` assina um JWT RS256 (`scope=spreadsheets`, validade 1h) e troca por um access token no `token_uri`
3. `MontarAbas` monta as abas a partir dos dados do mes atual
4. `EnviarAbas` le as abas existentes, cria as que faltam (`batchUpdate addSheet`), limpa (`values:batchClear`) e grava tudo (`values:batchUpdate`, `USER_ENTERED`)

**Abas geradas:** Resumo · Contas do mes · Receitas · Parcelamentos · Investimentos · Transacoes

**Quando roda:** ao salvar/ativar a integracao, no botao Testar e 1x por dia pelo agendador.

Erros 403/404 da API viram mensagens orientando a compartilhar a planilha / conferir o ID. `sheetsAPIBase` e `sheetsHTTP` sao variaveis para os testes apontarem para um servidor local.

## Contas do mes (internal/contas)

Resumo das contas do mes no formato do WhatsApp (blocos monoespacados ```` ``` ```` e `*negrito*`), com secoes **Fixas**, **Parcelamentos**, **Compras a vista** (parcelamentos 1/1) e **Resumo** (total, ja pago, a pagar, receitas e sobra).

| Funcao | Descricao |
|--------|-----------|
| `Texto(db, fid, mes)` | Carrega os dados e devolve o texto |
| `BuildTexto(...)` | Monta o texto a partir de dados ja carregados (testavel) |
| `TelegramHTML(txt)` | Converte para o HTML aceito pelo Telegram (`<b>`, `<pre>`) |
| `EmailHTML(txt)` | Corpo do e-mail no visual escuro do sistema |
| `Enviar(db, user, notif, mes, email, telegram)` | Envia pelos canais pedidos; devolve um `Resultado` por canal |
| `EnviarMensais(db, agora)` | Envio automatico do dia (a partir das 8h) |

**Envio sob demanda:** botoes **Telegram** e **E-mail** na janela WhatsApp de `/pagamentos` e em `/notificacoes` → `POST /pagamentos/enviar?mes=AAAA-MM&canal=telegram|email`.

**Envio mensal automatico** (`user_notificacoes`): cada usuario escolhe em `/notificacoes`:
- e-mail e/ou Telegram;
- **dia do envio** (1–31; em meses mais curtos sai no ultimo dia);
- **contas do mes atual ou do mes seguinte** (ex.: quem recebe dia 23 envia em 23/10 as contas de novembro);
- o proprio Chat ID do Telegram (vazio = chat da familia).

Cada usuario recebe no maximo uma vez por mes (`ultimo_envio`).

## Configuracao via UI

A pagina `/integracoes` exibe cards para cada integracao com toggle ativa/inativa, campos (KV), botoes **Testar** e **Salvar** e um botao **!** com "para que serve" e "como configurar".

**Segredos** (`bot_token`, `service_account_json`) nunca voltam para a pagina: o campo aparece vazio com "✓ configurado" e, se for salvo em branco, o valor atual e mantido (`valorParaSalvar`).

## Dashboard Widget

Quando ha dados coletados, o dashboard exibe os widgets "Indicadores" (Selic, CDI, IPCA, USD, EUR, BTC) e "Proximos feriados". A posicao/coluna de cada widget e configuravel pelo usuario (ver `07-templates.md`).
