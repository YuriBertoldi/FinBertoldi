# Testes — FinBertoldi

Suite em duas camadas: **unit** (rápidos, sem dependências) e **integration** (rodam contra PostgreSQL real).

## Testes Unitários

Sem banco. Rodam em < 1s.

```bash
go test ./...
```

### Cobertura por pacote

**`internal/handler`** (`handler_test.go`):
- `TestFormatBRL` — formatação monetária brasileira (separadores, negativo, arredondamento)
- `TestParseFloat` — parsing BRL → float64 (pontos de milhar, vírgula decimal)
- `TestParseInt` — parsing inteiro com trim
- `TestParseDate` — parsing YYYY-MM-DD com fallback para now
- `TestMesDisplay` — exibição "Janeiro/2026" a partir de "2026-01"
- `TestCalcResumo` — cards do dashboard: TotalDespesas, TotalPago, Sobra, **Caixa = Receitas − pago − investido/mês**
- `TestParcelaCalculoMesVigente` — lógica de parcela vigente por mês
- `TestJurosCompostosValorFuturo` — projeção FV (juros compostos mensais)
- `TestFireCalculo` — metas FIRE 25×/28.5×/33.3×
- `TestTaxaPoupanca` — cálculo de taxa de poupança com edge cases

- `TestResultadoEnvioHTML` — resultado do envio por canal, com o erro escapado (sem injeção de HTML)
- `TestMontarLayoutWidgets` / `TestMontarLayoutWidgetsMantemOsQueNaoVieram` — ordem, coluna e visibilidade dos widgets
- `TestCanaisEnvio` — `?canal=email|telegram`
- `TestValorParaSalvar` — segredo em branco mantém o valor salvo
- `TestValidarNovaSenha` — mínimo de 8 caracteres, diferente da atual, recusa senhas conhecidas

**`internal/contas`** (`texto_test.go`, `referencia_test.go`):
- `TestBuildTexto` — seções, totais, já pago/a pagar/sobra e ordenação por valor
- `TestWaLinhaLargura` / `TestWaValor` — alinhamento em 26 colunas e formatação BRL
- `TestTelegramHTML` / `TestEmailHTMLEscapa` — conversão para Telegram e e-mail com escape
- `TestMesReferencia` — mês atual/seguinte, inclusive virada de ano

**`internal/integrations`** (`telegram_test.go`, `sheets_test.go`):
- `TestValidarTokenTelegram` / `TestExplicarErroTelegram`
- `TestParseServiceAccount`, `TestAssinarJWT` (assinatura RS256 verificada com a chave pública), `TestTokenAcesso`, `TestExtrairIDPlanilha`, `TestMontarAbas`, `TestEnviarAbas`, `TestEnviarAbasErros` — Google Sheets contra um servidor simulado

**`internal/store`** (`store_test.go`):
- `TestHashSenhaERoundTrip` — bcrypt hash/verify round-trip
- `TestHashSenhaUnique` — salt aleatório (mesmo input → hashes diferentes)
- `TestOrdenarTopo` — widget fixo (resumo) sempre primeiro

**`internal/auth`** (`auth_test.go`):
- `TestGenerateToken` — token de 32 bytes hex (64 chars)
- `TestLerSMTPPadroesGmail` / `TestLerSMTPSemConfig` / `TestLerSMTPExplicito` — configuração de e-mail embutida
- `TestSetSessionCookie` — cookie com HttpOnly + SameSite
- `TestClearSessionCookie` — cookie com MaxAge=-1
- `TestCurrentUserContext` — leitura de user do contexto HTTP
- `TestCurrentUserSemContexto` — retorna nil sem contexto
- `TestProtectedRedirectsSemSessao` — redireciona para /login sem sessão
- `TestAdminOnlyBloqueiaUsuarioComum` — retorna 403 para não-admin

---

## Testes de Integração

Validam isolamento multi-tenant e comportamento real do banco.

### Pré-requisito

```bash
docker compose up postgres -d
```

### Rodando

```bash
go test -tags=integration ./...

# Banco externo
DATABASE_URL="postgres://user:pass@host:5432/db?sslmode=disable" \
  go test -tags=integration ./...
```

### O que é validado

**Isolamento multi-tenant** (vazamento = falha de segurança):
- Família B não lê dados da família A em despesas, receitas, investimentos, reserva, parcelamentos, empréstimos
- Família B não faz UPDATE/DELETE em registros da família A
- Família B não marca como pago despesa/empréstimo da família A

**Regras de negócio:**
- Parcelamentos auto-finalizam após total de parcelas
- Parcelamentos só aparecem no mês vigente
- Toggle pago em despesa e empréstimo
- `TestReservaEM_SaldoContaCorrente` — saldo = Σdepósitos − Σretiradas
- `TestReservaEM_Update` / `TestReservaEM_IsolamentoUpdate` — edição com isolamento multi-tenant
- `TestInvestimento_Update` / `TestInvestimento_IsolamentoUpdate` — edição de aportes
- `TestReceita_Update` / `TestReceita_IsolamentoUpdate` — edição de receitas
- `TestGetInvestidoNoMes` — soma de aportes + depósitos reserva EM no mês (base do card Caixa)

**Usuários e famílias:**
- Cadastro com família existente ou nova
- Mover usuário entre famílias
- Não permite excluir família com membros
- Renomear + contagem de membros

**Migração:**
- Idempotente (rodar várias vezes sem erro)
- Cria "Família Padrão" automaticamente
- Admin inicial usa `ADMIN_PASSWORD`; sem ela, gera senha aleatória (nunca `admin123`)

**Notificações, widgets e senha** (`notificacoes_integration_test.go`):
- Salvar/ler preferências; banco recusa dia fora de 1–31
- Quem recebe hoje: dia escolhido, último dia do mês (ex.: dia 31 em novembro), não repete no mesmo mês, ignora desligados e inativos
- Layout dos widgets: ordem, coluna inválida ignorada, widgets novos no fim
- `TrocarSenha` encerra as sessões em outros aparelhos e mantém a atual; `ResetSenha` encerra todas

**Google Sheets ponta a ponta** (`internal/integrations/sheets_integration_test.go`):
- Configuração no banco → token → abas criadas e gravadas, sem vazar dados de outra família

**Histórico e agregações:**
- Histórico de 6 meses com receitas recorrentes e despesas fixas

### Isolamento entre execuções

Cada execução cria um schema PostgreSQL temporário (`test_<timestamp>`) e o destrói no final. Não afeta dados reais.

---

## CI / Deploy (GitHub Actions)

`.github/workflows/deploy.yml` roda a cada push/PR na `main`:

1. `go build ./...` e `go test ./...` (unit)
2. Sobe um serviço **PostgreSQL 16** e roda `go test -tags=integration ./internal/store/ ./internal/integrations/` com `INTEGRATION_REQUIRED=1` — sem banco o teste **falha** em vez de ser pulado
3. Só se tudo passar (e for push na `main`): deploy via SSH (`git pull` + `docker compose up -d --build app pluggy-service`); as migrations rodam no startup do app

Localmente:

```bash
go build ./... && go test ./...
docker compose up postgres -d
go test -tags=integration ./...
```

## Cobertura

```bash
go test -cover ./...
go test -tags=integration -coverprofile=cover.out ./...
go tool cover -html=cover.out -o cover.html
```
