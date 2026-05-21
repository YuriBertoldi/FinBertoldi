-- ============================================================
-- FinBertoldi — Schema de referência (atualizado 2026-05)
-- Este arquivo é documentação. As migrations reais rodam via
-- runMigrations() em db.go ao subir a aplicação.
-- ============================================================

CREATE TABLE families (
    id        SERIAL PRIMARY KEY,
    nome      VARCHAR(120) UNIQUE NOT NULL,
    criado_em TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE users (
    id           SERIAL PRIMARY KEY,
    nome         VARCHAR(100) NOT NULL,
    email        VARCHAR(150) UNIQUE NOT NULL,
    senha_hash   TEXT NOT NULL,
    admin        BOOLEAN NOT NULL DEFAULT false,
    family_admin BOOLEAN NOT NULL DEFAULT false,
    ativo        BOOLEAN NOT NULL DEFAULT true,
    family_id    INTEGER NOT NULL REFERENCES families(id),
    criado_em    TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE sessions (
    token     TEXT PRIMARY KEY,
    user_id   INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    expira_em TIMESTAMPTZ NOT NULL,
    criado_em TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Categorias de despesa por familia (basica / cartao / vr)
CREATE TABLE categorias (
    id        SERIAL PRIMARY KEY,
    family_id INTEGER NOT NULL REFERENCES families(id),
    nome      VARCHAR(60) NOT NULL,
    grupo     VARCHAR(20) NOT NULL DEFAULT 'basica',
    cor       VARCHAR(7)  NOT NULL DEFAULT '#607d8b',
    ativo     BOOLEAN NOT NULL DEFAULT true,
    criado_em TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (family_id, nome)
);

-- Cartoes de credito por familia
CREATE TABLE cartoes (
    id        SERIAL PRIMARY KEY,
    family_id INTEGER NOT NULL REFERENCES families(id),
    nome      VARCHAR(60) NOT NULL,
    bandeira  VARCHAR(30) NOT NULL DEFAULT '',
    limite    NUMERIC(14,2) NOT NULL DEFAULT 0,
    cor       VARCHAR(7)  NOT NULL DEFAULT '#607d8b',
    ativo     BOOLEAN NOT NULL DEFAULT true,
    criado_em TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (family_id, nome)
);

-- Despesas fixas/recorrentes
CREATE TABLE despesas_fixas (
    id        SERIAL PRIMARY KEY,
    family_id INTEGER NOT NULL REFERENCES families(id),
    nome      VARCHAR(120) NOT NULL,
    valor     NUMERIC(14,2) NOT NULL DEFAULT 0,
    categoria VARCHAR(60) NOT NULL DEFAULT '',
    ativa     BOOLEAN NOT NULL DEFAULT true,
    criado_em TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Controle mensal de pagamento de cada despesa fixa
CREATE TABLE despesas_fixas_mes (
    id         SERIAL PRIMARY KEY,
    despesa_id INTEGER NOT NULL REFERENCES despesas_fixas(id) ON DELETE CASCADE,
    mes        DATE NOT NULL,
    pago       BOOLEAN NOT NULL DEFAULT false,
    pago_em    TIMESTAMPTZ,
    criado_em  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (despesa_id, mes)
);

-- Parcelamentos de cartao de credito
CREATE TABLE parcelamentos (
    id             SERIAL PRIMARY KEY,
    family_id      INTEGER NOT NULL REFERENCES families(id),
    descricao      VARCHAR(120) NOT NULL,
    cartao         VARCHAR(60) NOT NULL DEFAULT '',
    valor_parcela  NUMERIC(14,2) NOT NULL DEFAULT 0,
    parcela_atual  INTEGER NOT NULL DEFAULT 1,
    total_parcelas INTEGER NOT NULL DEFAULT 1,
    data_inicio    DATE NOT NULL,
    ativo          BOOLEAN NOT NULL DEFAULT true,
    criado_em      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Receitas (recorrentes e unicas)
CREATE TABLE receitas (
    id         SERIAL PRIMARY KEY,
    family_id  INTEGER NOT NULL REFERENCES families(id),
    descricao  VARCHAR(120) NOT NULL,
    valor      NUMERIC(14,2) NOT NULL DEFAULT 0,
    data       DATE NOT NULL,
    tipo       VARCHAR(60) NOT NULL DEFAULT '',
    recorrente BOOLEAN NOT NULL DEFAULT false,
    criado_em  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Investimentos (lancamentos mensais por instituicao/tipo)
CREATE TABLE investimentos (
    id          SERIAL PRIMARY KEY,
    family_id   INTEGER NOT NULL REFERENCES families(id),
    instituicao VARCHAR(100) NOT NULL,
    tipo        VARCHAR(60) NOT NULL DEFAULT '',
    valor       NUMERIC(14,2) NOT NULL DEFAULT 0,
    data        DATE NOT NULL,
    notas       TEXT NOT NULL DEFAULT '',
    criado_em   TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Reserva de emergencia (historico de saldo)
CREATE TABLE reserva_emergencia (
    id        SERIAL PRIMARY KEY,
    family_id INTEGER NOT NULL REFERENCES families(id),
    valor     NUMERIC(14,2) NOT NULL DEFAULT 0,
    data      DATE NOT NULL DEFAULT CURRENT_DATE,
    notas     TEXT NOT NULL DEFAULT '',
    criado_em TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Emprestimos (devo / emprestei)
CREATE TABLE emprestimos (
    id        SERIAL PRIMARY KEY,
    family_id INTEGER NOT NULL REFERENCES families(id),
    pessoa    VARCHAR(100) NOT NULL,
    valor     NUMERIC(14,2) NOT NULL DEFAULT 0,
    direcao   VARCHAR(20) NOT NULL DEFAULT 'devo',
    data      DATE NOT NULL DEFAULT CURRENT_DATE,
    pago      BOOLEAN NOT NULL DEFAULT false,
    pago_em   TIMESTAMPTZ,
    notas     TEXT NOT NULL DEFAULT '',
    criado_em TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Pagamentos parciais de emprestimos
CREATE TABLE emprestimo_pagamentos (
    id            SERIAL PRIMARY KEY,
    emprestimo_id INTEGER NOT NULL REFERENCES emprestimos(id) ON DELETE CASCADE,
    family_id     INTEGER NOT NULL REFERENCES families(id),
    valor         NUMERIC(14,2) NOT NULL DEFAULT 0,
    data          DATE NOT NULL DEFAULT CURRENT_DATE,
    notas         TEXT NOT NULL DEFAULT '',
    criado_em     TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Permissoes de tela por usuario (controle de acesso individual)
CREATE TABLE user_permissions (
    user_id   INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    tela      VARCHAR(50) NOT NULL,
    PRIMARY KEY (user_id, tela)
);

-- Controle de versao das migrations (gerenciado por runMigrations em db.go)
CREATE TABLE schema_migrations (
    version    INTEGER PRIMARY KEY,
    name       VARCHAR(200) NOT NULL,
    applied_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Indexes
CREATE INDEX idx_categorias_family         ON categorias(family_id);
CREATE INDEX idx_cartoes_family            ON cartoes(family_id);
CREATE INDEX idx_despesas_fixas_family     ON despesas_fixas(family_id);
CREATE INDEX idx_parcelamentos_family      ON parcelamentos(family_id);
CREATE INDEX idx_receitas_family           ON receitas(family_id);
CREATE INDEX idx_investimentos_family      ON investimentos(family_id);
CREATE INDEX idx_reserva_emergencia_family ON reserva_emergencia(family_id);
CREATE INDEX idx_emprestimos_family        ON emprestimos(family_id);
CREATE INDEX idx_emp_pagamentos_emprestimo ON emprestimo_pagamentos(emprestimo_id);
CREATE INDEX idx_emp_pagamentos_family     ON emprestimo_pagamentos(family_id);
CREATE INDEX idx_user_permissions_user     ON user_permissions(user_id);
