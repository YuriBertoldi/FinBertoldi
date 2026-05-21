# Testes — FinBertoldi

Suite de testes em duas camadas: **unit** (rápidos, sem dependências) e **integration** (rodam contra PostgreSQL real).

## Testes Unitários (rápidos)

Não precisam de banco. Rodam em <1s.

```bash
go test ./...
```

Cobertura:
- `helpers_test.go` — formatação BRL, parseFloat, parseInt, parseDate, mesDisplay, lógica de juros compostos, cálculos FIRE, vigência de parcelamentos
- `auth_test.go` — geração de tokens, hash bcrypt, cookies de sessão, middleware protected/adminOnly

## Testes de Integração

Validam o **isolamento multi-tenant** (crítico para segurança) e o comportamento real do banco.

### Pré-requisitos

PostgreSQL rodando (o `docker compose up postgres` do projeto já serve):

```bash
docker compose up postgres -d
```

### Rodando

```bash
go test -tags=integration ./...
```

Ou apontando para outro banco:

```bash
DATABASE_URL="postgres://user:pass@host:5432/db?sslmode=disable" \
  go test -tags=integration ./...
```

### O que é validado

**Isolamento entre famílias** (vazamento = falha grave de segurança):
- ✅ Despesas, receitas, investimentos, reserva, parcelamentos, empréstimos
- ✅ Família B não consegue UPDATE em registro da família A
- ✅ Família B não consegue DELETE de registro da família A
- ✅ Família B não consegue marcar como pago despesa da família A

**Regras de negócio:**
- ✅ Parcelamentos auto-finalizam após o total de parcelas
- ✅ Parcelamentos só aparecem no mês vigente (janela [início, início + total])
- ✅ Toggle pago em despesa/empréstimo

**Usuários e famílias:**
- ✅ Cadastro com família existente / nova
- ✅ Mover usuário entre famílias
- ✅ Não permite excluir família com membros
- ✅ Renomear e contagem de membros

**Migração:**
- ✅ Idempotente (rodar várias vezes sem erro)
- ✅ Cria "Família Padrão" automaticamente
- ✅ Admin padrão criado quando DB vazio

**Histórico e agregações:**
- ✅ Histórico de 6 meses inclui receitas recorrentes e despesas fixas

### Modo isolado

Cada teste cria um schema PostgreSQL temporário (`test_<timestamp>`) e o destrói no final, então **não interfere com seus dados reais**.

## CI / Pré-deploy

Para validar antes de qualquer deploy:

```bash
# 1. Build do binário (garante que compila)
go build ./...

# 2. Vet (lint estático)
go vet ./...

# 3. Testes unitários
go test ./...

# 4. Testes de integração (precisa do postgres rodando)
docker compose up postgres -d
go test -tags=integration ./...

# Se tudo passar → seguro pra deploy
docker compose up --build -d
```

## Cobertura

```bash
go test -cover ./...
go test -tags=integration -coverprofile=cover.out ./...
go tool cover -html=cover.out -o cover.html
```
