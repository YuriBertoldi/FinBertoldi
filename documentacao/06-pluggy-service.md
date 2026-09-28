# Modulo: pluggy-service

**Diretorio:** `pluggy-service/`
**Container:** separado do app principal (microservico)

## Responsabilidade

Integracao com a API da Pluggy (https://pluggy.ai) para sincronizacao automatica de transacoes bancarias. Roda como container separado para nao afetar o app principal.

## Estrutura

```
pluggy-service/
├── main.go              # HTTP server + sync scheduler
├── Dockerfile           # Multi-stage: golang:1.24 → alpine
├── go.mod / go.sum
├── pluggy/
│   ├── client.go        # REST client para API Pluggy
│   └── models.go        # Structs de request/response da API
└── sync/
    └── sync.go          # Logica de sincronizacao de transacoes
```

## Pacote: pluggy (client)

### Client

```go
type Client struct {
    clientID, clientSecret string
    httpClient             *http.Client
    apiKey                 string       // token de autenticacao (cache)
    expireAt               time.Time    // expiracao do token
}
```

O client autentica automaticamente na API Pluggy usando client_id/client_secret. O token (apiKey) e cacheado por 110 minutos (a API da 2h de validade).

### Metodos

| Metodo | Endpoint Pluggy | Descricao |
|--------|----------------|-----------|
| `authenticate()` | POST /auth | Autentica e cacheia token |
| `UpdateCredentials(id, secret)` | — | Atualiza credenciais dinamicamente |
| `CreateConnectToken(clientUserID)` | POST /connect_token | Token para Connect Widget |
| `GetItem(itemID)` | GET /items/{id} | Busca item conectado |
| `ListAccounts(itemID)` | GET /accounts?itemId={id} | Lista contas de um item |
| `ListTransactions(accountID, from, to)` | GET /transactions | Lista transacoes paginadas |
| `ListConnectors()` | GET /connectors | Lista conectores disponiveis |

### Models (API Pluggy)

| Struct | Descricao |
|--------|-----------|
| `AuthResponse` | Resposta de autenticacao (apiKey) |
| `Connector` | Instituicao financeira (nome, URL, tipo, pais) |
| `Item` | Conta conectada (id, connector, status, error) |
| `Account` | Conta bancaria (tipo, subtipo, saldo, moeda) |
| `Transaction` | Transacao (data, descricao, valor, categoria, providerCode) |
| `WebhookPayload` | Payload de webhook (event, id) |

## Pacote: sync

### Syncer

```go
type Syncer struct {
    db     *sql.DB
    client *pluggy.Client
}
```

### SyncAll()

Executado periodicamente pelo scheduler:
1. Lista todos os `pluggy_items` com status `active`
2. Para cada item, chama `syncItem()`
3. Atualiza `last_sync` e `status` no banco

### syncItem(familyID, itemID, connectorName)

1. Lista contas do item via API Pluggy
2. Para cada conta, busca transacoes dos ultimos 30 dias
3. Para cada transacao:
   - Gera `fit_id` (providerCode ou transaction ID)
   - Verifica se ja existe na tabela `transacoes_banco` (deduplicacao)
   - Se nao existe, insere com `origem='pluggy'` e `status='pendente'`
4. Retorna contagem de novas transacoes

### SyncItem(itemID)

Forca sync de um item especifico (chamado via endpoint ou webhook).

## HTTP Server (main.go)

### Endpoints

| Metodo | Rota | Descricao |
|--------|------|-----------|
| GET | `/api/pluggy/status` | Health check (`{"status":"ok"}`) |
| POST | `/api/pluggy/connect-token` | Gera token para Pluggy Connect Widget |
| POST | `/api/pluggy/webhook` | Recebe webhooks da Pluggy (item/created, item/updated) |
| POST | `/api/pluggy/items` | Registra item conectado + dispara sync inicial |
| POST | `/api/pluggy/sync/{item_id}` | Forca sync de um item |

### Credenciais

O servico tenta obter credenciais Pluggy de duas formas:
1. **Env vars:** `PLUGGY_CLIENT_ID`, `PLUGGY_CLIENT_SECRET`
2. **Banco de dados:** tabela `integracoes_config WHERE integracao='pluggy' AND ativo=true`

A opcao 2 permite que credenciais sejam configuradas pela UI do admin sem restart do container.

### Scheduler

Background goroutine que executa `SyncAll()` a cada `SYNC_INTERVAL` (default: 6h). Antes de cada sync, re-le credenciais do banco para pegar atualizacoes.

### Webhook

Quando a Pluggy envia webhook de `item/created` ou `item/updated`:
1. Busca o item na API para obter status atual
2. Atualiza `connector_name` e `status` na tabela `pluggy_items`
3. Se status e `UPDATED` ou `LOGIN_SUCCESS`, dispara sync em goroutine

## Variaveis de ambiente

| Variavel | Padrao | Descricao |
|----------|--------|-----------|
| `DATABASE_URL` | *(obrigatorio)* | String de conexao PostgreSQL |
| `PLUGGY_CLIENT_ID` | *(opcional)* | Client ID Pluggy (fallback: banco) |
| `PLUGGY_CLIENT_SECRET` | *(opcional)* | Client Secret Pluggy (fallback: banco) |
| `SYNC_INTERVAL` | `6h` | Intervalo do sync automatico |
| `PORT` | `8081` | Porta HTTP |
| `TZ` | `America/Sao_Paulo` | Timezone |
