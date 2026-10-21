# Modulo: store

**Pacote:** `internal/store`
**Arquivos:** um por dominio

| Arquivo | Conteudo |
|---|---|
| `conexao.go` | `NewDB` e registro do driver PostgreSQL (`lib/pq`) |
| `migrations.go` | Lista de migrations e `RunMigrations` |
| `categorias.go` · `cartoes.go` | Cadastros |
| `despesas.go` · `parcelamentos.go` | Despesas fixas, instancias mensais, parcelamentos e antecipacao |
| `receitas.go` · `historico.go` | Receitas e historico mensal do dashboard |
| `investimentos.go` · `reserva_emergencia.go` | Investimentos e reserva de emergencia |
| `emprestimos.go` | Emprestimos e pagamentos parciais |
| `autenticacao.go` | Hash de senha, sessoes e cadastro pendente via Google |
| `recuperacao_de_senha.go` | Tokens de redefinicao de senha |
| `usuarios.go` · `familias.go` · `permissoes_de_tela.go` | Usuarios, familias, permissoes e `EnsureAdmin` |
| `transacoes_bancarias.go` | Importacao, conversao e conciliacao de transacoes |
| `integracoes.go` · `integracao_pluggy.go` | Configuracao e chave-valor das integracoes, itens Pluggy |
| `dados_economicos.go` · `feriados.go` | Dados coletados pelas integracoes gratuitas |
| `dashboard_widgets.go` | Layout dos widgets do dashboard |
| `alertas_de_orcamento.go` | Quem pediu alerta de orcamento e quais faixas ja foram avisadas no mes |
| `regras_de_categorizacao.go` | Regras "descricao contem X" / "categoria do banco e X" -> categoria da familia; `AplicarRegrasDeCategorizacao` (so pendentes cuja categoria nao e da familia) |
| `orcamentos.go` | Orcamento mensal por categoria: `DefinirOrcamentoMensalDaCategoria` e `OrcamentosDoMes` (gasto = despesas fixas ativas + debitos do extrato no mes, sem transacoes convertidas ou ignoradas) |
| `notificacoes.go` | Envio mensal das contas |

Funcao nova vai no arquivo do dominio dela, sempre com `WHERE family_id = $N` quando o dado e da familia.

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

## Migrations (v1-v21)

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
| 13 | financiamento | Campos de financiamento em parcelamentos |
| 14 | password_resets | Tabela de tokens de reset de senha |
| 15 | integracoes_extras | Dados economicos, feriados, integracoes KV |
| 16 | dashboard_widgets | Visibilidade/ordem dos widgets do dashboard por usuario |
| 17 | user_notificacoes | Preferencias de envio mensal das contas (e-mail/Telegram, dia, mes atual/seguinte) |
| 18 | dashboard_widgets_coluna | Coluna (esquerda/direita) escolhida para cada widget |
| 19 | orcamento_mensal_por_categoria | Coluna `orcamento_mensal` em `categorias` (NULL = sem orcamento; CHECK > 0) |
| 20 | alerta_de_orcamento | Preferencias `alerta_orcamento_email`/`_telegram` em `user_notificacoes` e tabela `alertas_de_orcamento_enviados` (um registro por usuario, categoria, mes e faixa) |
| 21 | regras_de_categorizacao | Tabela `regras_de_categorizacao` e coluna `categoria_do_banco` em `transacoes_banco` (preenchida com a categoria atual quando ela nao e da familia) |

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

### Conciliacao Bancaria
- `VincularTransacaoDespesa(db, familyID, txnID, despesaID)` — vincula transacao a despesa existente
- `VincularTransacaoReceita(db, familyID, txnID, receitaID)` — vincula transacao a receita existente
- `DesvincularTransacao(db, familyID, txnID)` — remove vinculo
- `BuscarMatchesDespesas(db, familyID, valor, data)` — busca despesas compativeis
- `BuscarMatchesReceitas(db, familyID, valor, data)` — busca receitas compativeis
- `AutoMatchTransacoes(db, familyID)` — match automatico por valor/data

### Recuperacao de Senha
- `CreatePasswordReset(db, userID, token, expiry)` — cria token de reset
- `GetPasswordReset(db, token)` — busca token valido
- `UsePasswordReset(db, token)` — marca como usado
- `UpdateUserPassword(db, userID, hash)` — atualiza senha e encerra todas as sessoes do usuario
- `ResetSenha(db, id, hash)` — redefinicao por admin; encerra todas as sessoes
- `TrocarSenha(db, id, hash, manterToken)` — grava a senha e apaga as sessoes do usuario, exceto `manterToken` (sessao atual)
- `GetUserIDByEmail(db, email)` — busca user por email

### Integracoes KV (por familia)
- `SetIntegracaoKV(db, familyID, integracao, chave, valor)` — salva config
- `GetIntegracaoKV(db, familyID, integracao, chave)` — busca config
- `GetIntegracaoAtiva(db, familyID, integracao)` — verifica se ativa
- `SetIntegracaoAtiva(db, familyID, integracao, ativa)` — ativa/desativa

### Dados Economicos
- `UpsertDadoEconomico(db, tipo, valor, data, fonte)` — insere/atualiza indicador
- `GetUltimoDadoEconomico(db, tipo)` — ultimo valor de um tipo
- `GetDadosEconomicos(db)` — retorna DashboardIntegracoes com todos indicadores

### Feriados
- `UpsertFeriado(db, data, nome, tipo)` — insere feriado
- `GetFeriadosAno(db, ano)` — lista feriados do ano

### Notificacoes (contas do mes)
- `GetNotificacoes(db, userID)` — preferencias (padrao: dia 1, tudo desligado)
- `SaveNotificacoes(db, userID, n)` — UPSERT das preferencias
- `GetNotificacoesPendentes(db, hoje, mes)` — usuarios ativos com envio ligado cujo dia e hoje (ou o ultimo dia do mes, se o dia escolhido nao existir) e que ainda nao receberam no mes
- `MarcarEnvioMensal(db, userID, mes)` — registra o envio do mes

### Orcamento e alerta de orcamento
- `DefinirOrcamentoMensalDaCategoria(db, fid, categoriaID, limite)` — grava ou remove (nil) o limite
- `OrcamentosDoMes(db, fid, mes)` — gasto de cada categoria com orcamento: despesas fixas ativas + debitos do extrato no mes, sem transacoes `convertida` ou `ignorada`
- `UsuariosComAlertaDeOrcamento(db)` — usuarios ativos com alerta por e-mail ou Telegram ligado
- `AlertasDeOrcamentoEnviados(db, userID, mes)` / `RegistrarAlertaDeOrcamentoEnviado(...)` — o que ja foi avisado no mes, por categoria e faixa

O envio fica em `internal/contas/alerta_de_orcamento.go` (`EnviarAlertasDeOrcamento`), chamado pelo mesmo agendador de hora em hora do envio mensal, a partir das 8h. Cada faixa (80% e acima de 100%) sai uma vez por categoria e mes; ao estourar, o aviso de 80% e dado como enviado.

### Widgets do dashboard
- `GetDashboardLayout(db, userID)` — todos os widgets na ordem salva, com coluna e visibilidade; widgets nao salvos entram no fim com o padrao do `widgetCatalogo`
- `SaveDashboardLayout(db, userID, layout)` — grava ordem, coluna e visibilidade
- `OrdenarTopo(layout)` — mantem o widget fixo (`resumo`) sempre em primeiro
- `GetDashboardWidgets(db, userID)` — mapa de visibilidade (compatibilidade)

## Testes

- `store_test.go` — testes unitarios (sem banco)
- `integration_test.go` e `notificacoes_integration_test.go` — testes de integracao (build tag `integration`, requer PostgreSQL; rodam no CI). Detalhes em `TESTES.md`.
