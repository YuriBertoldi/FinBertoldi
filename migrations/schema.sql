-- Schema: Sistema de Controle Financeiro Pessoal
-- Baseado na planilha Fin-Bertoldi

-- Despesas fixas (contas básicas, cartão mensal, VR)
CREATE TABLE despesas_fixas (
    id        SERIAL PRIMARY KEY,
    nome      VARCHAR(200) NOT NULL,
    valor     NUMERIC(12,2) NOT NULL DEFAULT 0,
    categoria VARCHAR(50)  NOT NULL DEFAULT 'Básicas',
    -- Valores: 'Básicas', 'Cartão', 'VR'
    ativa     BOOLEAN NOT NULL DEFAULT true,
    criado_em TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Status mensal de cada despesa fixa (pago/não pago por mês)
CREATE TABLE despesas_fixas_mes (
    id         SERIAL PRIMARY KEY,
    despesa_id INTEGER NOT NULL REFERENCES despesas_fixas(id) ON DELETE CASCADE,
    mes        DATE NOT NULL,  -- sempre primeiro dia do mês: 2024-03-01
    pago       BOOLEAN NOT NULL DEFAULT false,
    pago_em    TIMESTAMPTZ,
    UNIQUE(despesa_id, mes)
);

-- Parcelamentos
CREATE TABLE parcelamentos (
    id             SERIAL PRIMARY KEY,
    descricao      VARCHAR(200) NOT NULL,
    cartao         VARCHAR(100) NOT NULL DEFAULT '',
    valor_parcela  NUMERIC(12,2) NOT NULL,
    parcela_atual  INTEGER NOT NULL DEFAULT 1,
    total_parcelas INTEGER NOT NULL,
    data_inicio    DATE NOT NULL DEFAULT CURRENT_DATE,
    ativo          BOOLEAN NOT NULL DEFAULT true,
    criado_em      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Receitas
CREATE TABLE receitas (
    id          SERIAL PRIMARY KEY,
    descricao   VARCHAR(200) NOT NULL,
    valor       NUMERIC(12,2) NOT NULL,
    data        DATE NOT NULL DEFAULT CURRENT_DATE,
    tipo        VARCHAR(100) NOT NULL DEFAULT 'Salário',
    recorrente  BOOLEAN NOT NULL DEFAULT false,
    criado_em   TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Investimentos
CREATE TABLE investimentos (
    id           SERIAL PRIMARY KEY,
    instituicao  VARCHAR(100) NOT NULL,  -- Rico, INCO, Previdência, etc.
    tipo         VARCHAR(100) NOT NULL DEFAULT 'Renda Variável',
    valor        NUMERIC(12,2) NOT NULL,
    data         DATE NOT NULL DEFAULT CURRENT_DATE,
    notas        TEXT NOT NULL DEFAULT '',
    criado_em    TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Reserva de Emergência (histórico de atualizações do saldo)
CREATE TABLE reserva_emergencia (
    id        SERIAL PRIMARY KEY,
    valor     NUMERIC(12,2) NOT NULL,
    data      DATE NOT NULL DEFAULT CURRENT_DATE,
    notas     TEXT NOT NULL DEFAULT '',
    criado_em TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Empréstimos
CREATE TABLE emprestimos (
    id        SERIAL PRIMARY KEY,
    pessoa    VARCHAR(100) NOT NULL,
    valor     NUMERIC(12,2) NOT NULL,
    direcao   VARCHAR(10) NOT NULL CHECK (direcao IN ('emprestei', 'devo')),
    data      DATE NOT NULL DEFAULT CURRENT_DATE,
    pago      BOOLEAN NOT NULL DEFAULT false,
    pago_em   TIMESTAMPTZ,
    notas     TEXT NOT NULL DEFAULT '',
    criado_em TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Dados de exemplo baseados na planilha
INSERT INTO despesas_fixas (nome, valor, categoria) VALUES
    ('Água', 0, 'Básicas'),
    ('Tel Yuri', 0, 'Básicas'),
    ('Tel Rhaizza', 0, 'Básicas'),
    ('Energia', 0, 'Básicas'),
    ('Internet', 0, 'Básicas'),
    ('Ração', 0, 'Básicas'),
    ('Gasolina', 0, 'Básicas'),
    ('Cabelo', 0, 'Básicas'),
    ('Agua mineral', 0, 'Básicas'),
    ('Academia', 0, 'Básicas'),
    ('Plano saúde Eloah/Rhaizza', 0, 'Básicas'),
    ('Previdência IR', 0, 'Básicas'),
    ('Netflix', 0, 'Cartão'),
    ('Amazon Prime', 0, 'Cartão'),
    ('Spotify', 0, 'Cartão'),
    ('Nubank Vida', 0, 'Cartão'),
    ('Seg. Vida Rhaizza', 0, 'Cartão'),
    ('Natação', 0, 'Cartão'),
    ('Claude Code', 0, 'Cartão'),
    ('HBO Max', 0, 'Cartão');
