# Banco de Dados

**SGBD:** PostgreSQL 16 Alpine
**Nome:** fincontrol
**Schema:** gerenciado via migrations idempotentes no startup

## Tabelas

### families
Familia (grupo de usuarios que compartilham dados).

| Coluna | Tipo | Descricao |
|--------|------|-----------|
| id | SERIAL PK | ID |
| nome | VARCHAR(200) | Nome da familia |
| criado_em | TIMESTAMPTZ | Data de criacao |

### users
Usuarios do sistema.

| Coluna | Tipo | Descricao |
|--------|------|-----------|
| id | SERIAL PK | ID |
| nome | VARCHAR(200) | Nome |
| email | VARCHAR(200) UNIQUE | Email (login) |
| senha_hash | TEXT | Hash bcrypt |
| admin | BOOLEAN | Admin global |
| family_admin | BOOLEAN | Admin da familia |
| ativo | BOOLEAN | Usuario ativo |
| family_id | INTEGER FK families | Familia |
| criado_em | TIMESTAMPTZ | Data de criacao |

### sessions
Sessoes de autenticacao.

| Coluna | Tipo | Descricao |
|--------|------|-----------|
| token | VARCHAR(200) PK | Token hex 64 chars |
| user_id | INTEGER FK users | Usuario |
| expires_at | TIMESTAMPTZ | Expiracao (30 dias) |

### categorias
Categorias de despesas por familia.

| Coluna | Tipo | Descricao |
|--------|------|-----------|
| id | SERIAL PK | ID |
| family_id | INTEGER FK families | Familia |
| nome | VARCHAR(200) | Nome |
| grupo | VARCHAR(50) | basica, cartao, vr |
| cor | VARCHAR(20) | Cor hex |
| ativo | BOOLEAN | Ativa |
| criado_em | TIMESTAMPTZ | Data de criacao |

### cartoes
Cartoes de credito/debito.

| Coluna | Tipo | Descricao |
|--------|------|-----------|
| id | SERIAL PK | ID |
| family_id | INTEGER FK families | Familia |
| nome | VARCHAR(200) | Nome do cartao |
| bandeira | VARCHAR(50) | Visa, Mastercard, etc |
| limite | NUMERIC(12,2) | Limite do cartao |
| cor | VARCHAR(20) | Cor hex |
| ativo | BOOLEAN | Ativo |
| criado_em | TIMESTAMPTZ | Data de criacao |

### despesas_fixas
Despesas recorrentes.

| Coluna | Tipo | Descricao |
|--------|------|-----------|
| id | SERIAL PK | ID |
| family_id | INTEGER FK families | Familia |
| nome | VARCHAR(200) | Nome da despesa |
| valor | NUMERIC(12,2) | Valor mensal |
| categoria | VARCHAR(100) | Categoria |
| grupo | VARCHAR(50) | basica, cartao, vr |
| ativa | BOOLEAN | Ativa |
| criado_em | TIMESTAMPTZ | Data de criacao |

### despesas_fixas_mes
Instancias mensais de despesas fixas (controle pago/nao pago).

| Coluna | Tipo | Descricao |
|--------|------|-----------|
| id | SERIAL PK | ID |
| despesa_id | INTEGER FK despesas_fixas | Despesa fixa |
| mes | VARCHAR(7) | "YYYY-MM" |
| pago | BOOLEAN | Pago neste mes |

### parcelamentos
Compras parceladas.

| Coluna | Tipo | Descricao |
|--------|------|-----------|
| id | SERIAL PK | ID |
| family_id | INTEGER FK families | Familia |
| descricao | VARCHAR(300) | Descricao |
| cartao | VARCHAR(100) | Cartao utilizado |
| valor_parcela | NUMERIC(12,2) | Valor de cada parcela |
| parcela_atual | INTEGER | Parcela atual |
| total_parcelas | INTEGER | Total de parcelas |
| data_inicio | DATE | Data da primeira parcela |
| criado_em | TIMESTAMPTZ | Data de criacao |

### receitas
Receitas (salario, freelance, etc).

| Coluna | Tipo | Descricao |
|--------|------|-----------|
| id | SERIAL PK | ID |
| family_id | INTEGER FK families | Familia |
| descricao | VARCHAR(300) | Descricao |
| valor | NUMERIC(12,2) | Valor |
| data | DATE | Data |
| tipo | VARCHAR(50) | Tipo (salario, freelance, etc) |
| recorrente | BOOLEAN | Se repete todo mes |
| criado_em | TIMESTAMPTZ | Data de criacao |

### investimentos
Aportes de investimento.

| Coluna | Tipo | Descricao |
|--------|------|-----------|
| id | SERIAL PK | ID |
| family_id | INTEGER FK families | Familia |
| instituicao | VARCHAR(200) | Instituicao (XP, Rico, etc) |
| tipo | VARCHAR(100) | Tipo (CDB, Acoes, FII, etc) |
| valor | NUMERIC(12,2) | Valor do aporte |
| data | DATE | Data do aporte |
| notas | TEXT | Observacoes |
| criado_em | TIMESTAMPTZ | Data de criacao |

### reserva_emergencia
Lancamentos da reserva de emergencia (depositos e retiradas).

| Coluna | Tipo | Descricao |
|--------|------|-----------|
| id | SERIAL PK | ID |
| family_id | INTEGER FK families | Familia |
| valor | NUMERIC(12,2) | Valor (positivo=deposito, negativo=retirada) |
| data | DATE | Data |
| tipo | VARCHAR(20) | deposito ou retirada |
| notas | TEXT | Observacoes |
| criado_em | TIMESTAMPTZ | Data de criacao |

### emprestimos
Emprestimos (devo ou me devem).

| Coluna | Tipo | Descricao |
|--------|------|-----------|
| id | SERIAL PK | ID |
| family_id | INTEGER FK families | Familia |
| pessoa | VARCHAR(200) | Pessoa |
| valor | NUMERIC(12,2) | Valor original |
| direcao | VARCHAR(20) | devo ou me_devem |
| data | DATE | Data |
| pago | BOOLEAN | Quitado |
| notas | TEXT | Observacoes |
| criado_em | TIMESTAMPTZ | Data de criacao |

### emprestimos_pagamentos
Pagamentos parciais de emprestimos.

| Coluna | Tipo | Descricao |
|--------|------|-----------|
| id | SERIAL PK | ID |
| emprestimo_id | INTEGER FK emprestimos | Emprestimo |
| valor | NUMERIC(12,2) | Valor do pagamento |
| data | DATE | Data do pagamento |
| criado_em | TIMESTAMPTZ | Data de criacao |

### screen_permissions
Controle de acesso por tela (telas bloqueadas por usuario).

| Coluna | Tipo | Descricao |
|--------|------|-----------|
| id | SERIAL PK | ID |
| user_id | INTEGER FK users | Usuario |
| screen | VARCHAR(50) | Identificador da tela |

### transacoes_banco
Transacoes importadas de extratos bancarios.

| Coluna | Tipo | Descricao |
|--------|------|-----------|
| id | SERIAL PK | ID |
| family_id | INTEGER FK families | Familia |
| data | DATE | Data da transacao |
| descricao | TEXT | Descricao |
| valor | NUMERIC(12,2) | Valor (negativo=debito, positivo=credito) |
| tipo | VARCHAR(20) | debito ou credito |
| categoria | VARCHAR(100) | Categoria (se atribuida) |
| origem | VARCHAR(20) | csv, ofx, pluggy |
| banco | VARCHAR(200) | Nome do banco |
| fit_id | VARCHAR(200) | ID unico para deduplicacao |
| status | VARCHAR(20) | pendente, categorizada, convertida, ignorada |
| despesa_id | INTEGER | FK para despesa criada (se convertida) |
| receita_id | INTEGER | FK para receita criada (se convertida) |
| pluggy_item_id | VARCHAR(255) | ID do item Pluggy (se origem=pluggy) |
| criado_em | TIMESTAMPTZ | Data de importacao |

**Indices:** `idx_transacoes_banco_family`, `idx_transacoes_banco_fit_id`

### pluggy_items
Contas bancarias conectadas via Pluggy.

| Coluna | Tipo | Descricao |
|--------|------|-----------|
| id | SERIAL PK | ID |
| family_id | INTEGER FK families | Familia |
| item_id | VARCHAR(255) UNIQUE | ID do item na Pluggy |
| connector_name | VARCHAR(200) | Nome do banco |
| status | VARCHAR(50) | Status (active, error, etc) |
| last_sync | TIMESTAMPTZ | Data da ultima sincronizacao |
| criado_em | TIMESTAMPTZ | Data de conexao |

**Indice:** `idx_pluggy_items_family`

### integracoes_config
Configuracao de integracoes externas (generica).

| Coluna | Tipo | Descricao |
|--------|------|-----------|
| id | SERIAL PK | ID |
| integracao | VARCHAR(50) UNIQUE | Nome da integracao (pluggy, belvo, etc) |
| client_id | TEXT | Client ID da API |
| client_secret | TEXT | Client Secret da API |
| service_url | TEXT | URL do microservico |
| ativo | BOOLEAN | Integracao ativa |
| atualizado_em | TIMESTAMPTZ | Data da ultima atualizacao |

**Seed:** linha padrao com `integracao='pluggy'` e `service_url='http://pluggy-service:8081'`
