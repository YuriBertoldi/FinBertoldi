# Modulo: models

**Pacote:** `internal/models`
**Arquivo:** `models.go`

## Responsabilidade

Define todas as structs de dominio (entidades do banco) e structs de dados de pagina (page data para os templates).

## Entidades de dominio

| Struct | Tabela | Descricao |
|--------|--------|-----------|
| `DespesaFixa` | `despesas_fixas` | Despesa recorrente (aluguel, internet, etc) |
| `DespesaMes` | `despesas_fixas_mes` | Instancia mensal de uma despesa fixa com flag `Pago` |
| `Categoria` | `categorias` | Categoria de despesa (nome, grupo, cor) |
| `Cartao` | `cartoes` | Cartao de credito/debito |
| `Parcelamento` | `parcelamentos` | Compra parcelada em cartao |
| `Receita` | `receitas` | Receita (salario, freelance, etc) |
| `Investimento` | `investimentos` | Aporte de investimento |
| `ReservaEM` | `reserva_emergencia` | Lancamento na reserva de emergencia |
| `Emprestimo` | `emprestimos` | Emprestimo (devo ou me devem) |
| `User` | `users` | Usuario com flags Admin, FamilyAdmin, Ativo |
| `Family` | `families` | Familia com contagem de membros |
| `TransacaoBanco` | `transacoes_banco` | Transacao importada de extrato bancario |
| `PluggyItem` | `pluggy_items` | Conta bancaria conectada via Pluggy |
| `IntegracaoConfig` | `integracoes_config` | Configuracao de integracao externa (Pluggy, etc) |

## Grupos de categoria

O campo `Grupo` de `Categoria` determina como a despesa aparece no dashboard:
- `basica` — Contas basicas (aluguel, internet, etc)
- `cartao` — Cartao de credito
- `vr` — VR/Alimentacao

## Page Data (structs de template)

| Struct | Template | Campos principais |
|--------|----------|-------------------|
| `BasePage` | todos | CurrentUser, Active (sidebar), Title, BlockedScreens |
| `DashboardData` | dashboard | Resumo, Despesas por grupo, Parcelamentos, Historico 6 meses |
| `DashboardResumo` | dashboard cards | TotalReceitas, TotalDespesas, Sobra, Caixa, TotalInvestido |
| `DespesasPage` | despesas | Despesas, Parcelamentos, Categorias, Cartoes, mes navegavel |
| `ReceitasPage` | receitas | Lista de Receitas |
| `InvestimentosPage` | investimentos | Investimentos, Totais por tipo, ReservaEM |
| `EmprestimosPage` | emprestimos | Emprestimos, TotalDevo, TotalAReceber |
| `CadastrosPage` | cadastros | Categorias, Cartoes, PluggyConfig, PluggyContas, PluggyStatus |
| `TransacoesPage` | transacoes | Transacoes, Filtros, Resultado importacao |
| `PlanejamentoPage` | planejamento | Dados FIRE, taxa poupanca, patrimonio |
| `UsuariosPage` | usuarios | Lista de usuarios + familias |
| `MeuTimePage` | minha-familia | Membros da familia, Telas para controle de acesso |
| `ImportExportPage` | importexport | Resultado de importacao XLSX |

## Status de TransacaoBanco

Uma transacao bancaria importada passa pelos seguintes status:
- `pendente` — recem importada, aguardando acao do usuario
- `categorizada` — usuario atribuiu uma categoria
- `convertida` — transformada em despesa ou receita real
- `ignorada` — descartada pelo usuario

## Tipo de TransacaoBanco

- `debito` — valor negativo (saida)
- `credito` — valor positivo (entrada)

## Origem de TransacaoBanco

- `csv` — importado via arquivo CSV
- `ofx` — importado via arquivo OFX
- `pluggy` — sincronizado automaticamente via Pluggy API
