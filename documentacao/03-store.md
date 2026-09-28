# Modulo: store

**Pacote:** `internal/store`
**Arquivo:** `store.go`

## Responsabilidade

Camada de acesso a dados. Todas as queries SQL do sistema estao neste pacote. Tambem responsavel por:
- Conexao com o banco (`NewDB`)
- Migrations idempotentes (`RunMigrations`)
- Criacao do admin padrao (`EnsureAdmin`)
- Hash de senhas (`HashSenha`)

## Conexao

`NewDB()` le a connection string de:
1. `DATABASE_URL` (prioridade, usado em producao)
2. Variaveis individuais: `DB_HOST`, `DB_PORT`, `DB_USER`, `DB_PASS`, `DB_NAME`

## Migrations (v1-v12)

Executadas automaticamente no startup via `RunMigrations()`. Cada migration e registrada na tabela `schema_migrations` (version + nome + data de aplicacao). Idempotentes: se ja aplicada, e ignorada.

| Versao | Nome | Descricao |
|--------|------|-----------|
| 1 | families_and_users | Tabelas base: families, users |
| 2 | sessions | Sessoes de autenticacao |
| 3 | categorias | Categorias de despesa |
| 4 | despesas_fixas | Despesas fixas + instancias mensais |
| 5 | parcelamentos | Parcelamentos de cartao |
| 6 | receitas | Receitas |
| 7 | investimentos | Investimentos + reserva de emergencia |
| 8 | emprestimos | Emprestimos + pagamentos parciais |
| 9 | cartoes | Cartoes de credito |
| 10 | screen_permissions | Controle de acesso por tela |
| 11 | transacoes_banco_e_pluggy_items | Transacoes bancarias + itens Pluggy |
| 12 | integracoes_config | Configuracao de integracoes externas |

## Funcoes por dominio

### Categorias
- `GetCategorias(db, familyID)` — lista todas
- `CreateCategoria(db, familyID, nome, grupo, cor)` — cria nova
- `UpdateCategoria(db, familyID, id, nome, grupo, cor)` — atualiza
- `DeleteCategoria(db, familyID, id)` — remove
- `ToggleCategoriaAtiva(db, familyID, id)` — ativa/desativa

### Cartoes
- `GetCartoes(db, familyID)` — lista todos
- `CreateCartao(db, familyID, nome, bandeira, limite, cor)` — cria novo
- `UpdateCartao(...)` — atualiza
- `DeleteCartao(db, familyID, id)` — remove
- `ToggleCartaoAtivo(db, familyID, id)` — ativa/desativa

### Despesas Fixas
- `GetDespesasFixas(db, familyID)` — lista ativas
- `CreateDespesaFixa(...)` — cria nova
- `UpdateDespesaFixa(...)` — atualiza
- `DeleteDespesaFixa(db, familyID, id)` — remove
- `GetDespesasDoMes(db, familyID, mes, grupo)` — instancias de um mes/grupo
- `TogglePagoDespesa(db, familyID, id)` — marca pago/nao pago

### Parcelamentos
- `GetParcelamentos(db, familyID)` — lista todos
- `CreateParcelamento(...)` — cria novo
- `GetParcelamentosDoMes(db, familyID, mes)` — parcelas do mes
- `TogglePagoParcelamento(db, familyID, id)` — marca pago

### Receitas
- `GetReceitas(db, familyID, mes)` — lista do mes
- `GetReceitasMes(db, familyID, mes)` — total do mes
- `CreateReceita(...)` — cria nova

### Investimentos
- `GetInvestimentos(db, familyID)` — lista todos
- `GetInvestidoNoMes(db, familyID, mes)` — soma do mes
- `GetTotalInvestido(db, familyID)` — soma geral

### Emprestimos
- `GetEmprestimos(db, familyID)` — lista todos
- `CreateEmprestimo(...)` — cria novo
- `PagamentoEmprestimo(db, familyID, id, valor)` — quitacao parcial

### Transacoes Bancarias
- `CreateTransacaoBancoBatch(db, familyID, txns)` — insere em lote com dedup por fit_id
- `GetTransacoesBanco(db, familyID, mes, origem, status, banco)` — lista filtrada
- `ConverterTransacaoEmDespesa(db, familyID, id, categoria)` — cria despesa a partir da transacao
- `ConverterTransacaoEmReceita(db, familyID, id, tipo)` — cria receita a partir da transacao
- `IgnorarTransacao(db, familyID, id)` — marca como ignorada
- `CategorizarTransacao(db, familyID, id, categoria)` — atribui categoria

### Pluggy Items
- `GetPluggyItems(db, familyID)` — lista contas conectadas
- `CreatePluggyItem(db, familyID, itemID, connectorName)` — registra nova conta
- `UpdatePluggyItemSync(db, itemID)` — atualiza timestamp de sync
- `DeletePluggyItem(db, familyID, itemID)` — remove conta

### Integracoes Config
- `GetIntegracaoConfig(db, integracao)` — busca config por nome (ex: "pluggy")
- `SaveIntegracaoConfig(db, config)` — UPSERT de configuracao

### Usuarios e Familias
- `GetUserByEmail(db, email)` — busca por email
- `GetUserBySession(db, token)` — busca por sessao
- `CreateSession(db, userID)` — cria sessao
- `GetBlockedScreens(db, userID)` — telas bloqueadas

## Testes

- `store_test.go` — testes unitarios (sem banco)
- `integration_test.go` — testes de integracao (build tag: `integration`, requer postgres rodando)
