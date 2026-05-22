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

**`internal/store`** (`store_test.go`):
- `TestHashSenhaERoundTrip` — bcrypt hash/verify round-trip
- `TestHashSenhaUnique` — salt aleatório (mesmo input → hashes diferentes)
- `TestGetEnv` — variável de ambiente com fallback

**`internal/auth`** (`auth_test.go`):
- `TestGenerateToken` — token de 32 bytes hex (64 chars)
- `TestGenerateTokenUnique` — tokens únicos por chamada
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
- Admin padrão criado quando DB vazio

**Histórico e agregações:**
- Histórico de 6 meses com receitas recorrentes e despesas fixas

### Isolamento entre execuções

Cada execução cria um schema PostgreSQL temporário (`test_<timestamp>`) e o destrói no final. Não afeta dados reais.

---

## CI / Pré-deploy

```bash
go build ./...                          # garante que compila
go vet ./...                            # lint estático
go test ./...                           # unit
docker compose up postgres -d
go test -tags=integration ./...         # integração
docker compose up --build -d            # se tudo passou → deploy
```

## Cobertura

```bash
go test -cover ./...
go test -tags=integration -coverprofile=cover.out ./...
go tool cover -html=cover.out -o cover.html
```
