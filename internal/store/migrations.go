package store

import (
	"database/sql"
	"fmt"
	"log"
)

type migration struct {
	version int
	name    string
	stmts   []string
}

var migrations = []migration{
	{
		version: 1,
		name:    "core_tables",
		stmts: []string{
			`CREATE TABLE IF NOT EXISTS families (
				id        SERIAL PRIMARY KEY,
				nome      VARCHAR(120) UNIQUE NOT NULL,
				criado_em TIMESTAMPTZ NOT NULL DEFAULT NOW()
			)`,
			`CREATE TABLE IF NOT EXISTS users (
				id         SERIAL PRIMARY KEY,
				nome       VARCHAR(100) NOT NULL,
				email      VARCHAR(150) UNIQUE NOT NULL,
				senha_hash TEXT NOT NULL,
				admin      BOOLEAN NOT NULL DEFAULT false,
				ativo      BOOLEAN NOT NULL DEFAULT true,
				family_id  INTEGER REFERENCES families(id),
				criado_em  TIMESTAMPTZ NOT NULL DEFAULT NOW()
			)`,
			`CREATE TABLE IF NOT EXISTS sessions (
				token     TEXT PRIMARY KEY,
				user_id   INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
				expira_em TIMESTAMPTZ NOT NULL,
				criado_em TIMESTAMPTZ NOT NULL DEFAULT NOW()
			)`,
		},
	},
	{
		version: 2,
		name:    "aux_tables",
		stmts: []string{
			`CREATE TABLE IF NOT EXISTS categorias (
				id        SERIAL PRIMARY KEY,
				family_id INTEGER NOT NULL REFERENCES families(id),
				nome      VARCHAR(60) NOT NULL,
				grupo     VARCHAR(20) NOT NULL DEFAULT 'basica',
				cor       VARCHAR(7)  NOT NULL DEFAULT '#607d8b',
				ativo     BOOLEAN NOT NULL DEFAULT true,
				criado_em TIMESTAMPTZ NOT NULL DEFAULT NOW(),
				UNIQUE (family_id, nome)
			)`,
			`CREATE TABLE IF NOT EXISTS cartoes (
				id        SERIAL PRIMARY KEY,
				family_id INTEGER NOT NULL REFERENCES families(id),
				nome      VARCHAR(60) NOT NULL,
				bandeira  VARCHAR(30) NOT NULL DEFAULT '',
				limite    NUMERIC(14,2) NOT NULL DEFAULT 0,
				cor       VARCHAR(7)  NOT NULL DEFAULT '#607d8b',
				ativo     BOOLEAN NOT NULL DEFAULT true,
				criado_em TIMESTAMPTZ NOT NULL DEFAULT NOW(),
				UNIQUE (family_id, nome)
			)`,
		},
	},
	{
		version: 3,
		name:    "finance_tables",
		stmts: []string{
			`CREATE TABLE IF NOT EXISTS despesas_fixas (
				id        SERIAL PRIMARY KEY,
				family_id INTEGER NOT NULL REFERENCES families(id),
				nome      VARCHAR(120) NOT NULL,
				valor     NUMERIC(14,2) NOT NULL DEFAULT 0,
				categoria VARCHAR(60) NOT NULL DEFAULT '',
				ativa     BOOLEAN NOT NULL DEFAULT true,
				criado_em TIMESTAMPTZ NOT NULL DEFAULT NOW()
			)`,
			`CREATE TABLE IF NOT EXISTS despesas_fixas_mes (
				id         SERIAL PRIMARY KEY,
				despesa_id INTEGER NOT NULL REFERENCES despesas_fixas(id) ON DELETE CASCADE,
				mes        DATE NOT NULL,
				pago       BOOLEAN NOT NULL DEFAULT false,
				pago_em    TIMESTAMPTZ,
				criado_em  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
				UNIQUE (despesa_id, mes)
			)`,
			`CREATE TABLE IF NOT EXISTS parcelamentos (
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
			)`,
			`CREATE TABLE IF NOT EXISTS receitas (
				id         SERIAL PRIMARY KEY,
				family_id  INTEGER NOT NULL REFERENCES families(id),
				descricao  VARCHAR(120) NOT NULL,
				valor      NUMERIC(14,2) NOT NULL DEFAULT 0,
				data       DATE NOT NULL,
				tipo       VARCHAR(60) NOT NULL DEFAULT '',
				recorrente BOOLEAN NOT NULL DEFAULT false,
				criado_em  TIMESTAMPTZ NOT NULL DEFAULT NOW()
			)`,
			`CREATE TABLE IF NOT EXISTS investimentos (
				id          SERIAL PRIMARY KEY,
				family_id   INTEGER NOT NULL REFERENCES families(id),
				instituicao VARCHAR(100) NOT NULL,
				tipo        VARCHAR(60) NOT NULL DEFAULT '',
				valor       NUMERIC(14,2) NOT NULL DEFAULT 0,
				data        DATE NOT NULL,
				notas       TEXT NOT NULL DEFAULT '',
				criado_em   TIMESTAMPTZ NOT NULL DEFAULT NOW()
			)`,
			`CREATE TABLE IF NOT EXISTS reserva_emergencia (
				id        SERIAL PRIMARY KEY,
				family_id INTEGER NOT NULL REFERENCES families(id),
				valor     NUMERIC(14,2) NOT NULL DEFAULT 0,
				data      DATE NOT NULL DEFAULT CURRENT_DATE,
				notas     TEXT NOT NULL DEFAULT '',
				criado_em TIMESTAMPTZ NOT NULL DEFAULT NOW()
			)`,
			`CREATE TABLE IF NOT EXISTS emprestimos (
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
			)`,
		},
	},
	{
		version: 4,
		name:    "loans_and_permissions",
		stmts: []string{
			`CREATE TABLE IF NOT EXISTS emprestimo_pagamentos (
				id            SERIAL PRIMARY KEY,
				emprestimo_id INTEGER NOT NULL REFERENCES emprestimos(id) ON DELETE CASCADE,
				family_id     INTEGER NOT NULL REFERENCES families(id),
				valor         NUMERIC(14,2) NOT NULL DEFAULT 0,
				data          DATE NOT NULL DEFAULT CURRENT_DATE,
				notas         TEXT NOT NULL DEFAULT '',
				criado_em     TIMESTAMPTZ NOT NULL DEFAULT NOW()
			)`,
			`CREATE TABLE IF NOT EXISTS user_permissions (
				user_id   INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
				tela      VARCHAR(50) NOT NULL,
				PRIMARY KEY (user_id, tela)
			)`,
		},
	},
	{
		version: 5,
		name:    "indexes",
		stmts: []string{
			`CREATE INDEX IF NOT EXISTS idx_categorias_family          ON categorias(family_id)`,
			`CREATE INDEX IF NOT EXISTS idx_cartoes_family             ON cartoes(family_id)`,
			`CREATE INDEX IF NOT EXISTS idx_despesas_fixas_family      ON despesas_fixas(family_id)`,
			`CREATE INDEX IF NOT EXISTS idx_parcelamentos_family       ON parcelamentos(family_id)`,
			`CREATE INDEX IF NOT EXISTS idx_receitas_family            ON receitas(family_id)`,
			`CREATE INDEX IF NOT EXISTS idx_investimentos_family       ON investimentos(family_id)`,
			`CREATE INDEX IF NOT EXISTS idx_reserva_emergencia_family  ON reserva_emergencia(family_id)`,
			`CREATE INDEX IF NOT EXISTS idx_emprestimos_family         ON emprestimos(family_id)`,
			`CREATE INDEX IF NOT EXISTS idx_emp_pagamentos_emprestimo  ON emprestimo_pagamentos(emprestimo_id)`,
			`CREATE INDEX IF NOT EXISTS idx_emp_pagamentos_family      ON emprestimo_pagamentos(family_id)`,
			`CREATE INDEX IF NOT EXISTS idx_user_permissions_user      ON user_permissions(user_id)`,
		},
	},
	{
		version: 6,
		name:    "backcompat_columns",
		stmts: []string{
			`ALTER TABLE users              ADD COLUMN IF NOT EXISTS family_id    INTEGER REFERENCES families(id)`,
			`ALTER TABLE users              ADD COLUMN IF NOT EXISTS family_admin BOOLEAN NOT NULL DEFAULT false`,
			`ALTER TABLE despesas_fixas     ADD COLUMN IF NOT EXISTS family_id   INTEGER REFERENCES families(id)`,
			`ALTER TABLE parcelamentos      ADD COLUMN IF NOT EXISTS family_id   INTEGER REFERENCES families(id)`,
			`ALTER TABLE receitas           ADD COLUMN IF NOT EXISTS family_id   INTEGER REFERENCES families(id)`,
			`ALTER TABLE investimentos      ADD COLUMN IF NOT EXISTS family_id   INTEGER REFERENCES families(id)`,
			`ALTER TABLE reserva_emergencia ADD COLUMN IF NOT EXISTS family_id   INTEGER REFERENCES families(id)`,
			`ALTER TABLE emprestimos        ADD COLUMN IF NOT EXISTS family_id   INTEGER REFERENCES families(id)`,
			`ALTER TABLE emprestimos        ADD COLUMN IF NOT EXISTS pago_em     TIMESTAMPTZ`,
		},
	},
	{
		version: 7,
		name:    "default_family_and_backfill",
		stmts: []string{
			`INSERT INTO families (nome) VALUES ('Família Padrão') ON CONFLICT (nome) DO NOTHING`,
			`UPDATE users              SET family_id = (SELECT id FROM families WHERE nome='Família Padrão') WHERE family_id IS NULL`,
			`UPDATE despesas_fixas     SET family_id = (SELECT id FROM families WHERE nome='Família Padrão') WHERE family_id IS NULL`,
			`UPDATE parcelamentos      SET family_id = (SELECT id FROM families WHERE nome='Família Padrão') WHERE family_id IS NULL`,
			`UPDATE receitas           SET family_id = (SELECT id FROM families WHERE nome='Família Padrão') WHERE family_id IS NULL`,
			`UPDATE investimentos      SET family_id = (SELECT id FROM families WHERE nome='Família Padrão') WHERE family_id IS NULL`,
			`UPDATE reserva_emergencia SET family_id = (SELECT id FROM families WHERE nome='Família Padrão') WHERE family_id IS NULL`,
			`UPDATE emprestimos        SET family_id = (SELECT id FROM families WHERE nome='Família Padrão') WHERE family_id IS NULL`,
			`ALTER TABLE users              ALTER COLUMN family_id SET NOT NULL`,
			`ALTER TABLE despesas_fixas     ALTER COLUMN family_id SET NOT NULL`,
			`ALTER TABLE parcelamentos      ALTER COLUMN family_id SET NOT NULL`,
			`ALTER TABLE receitas           ALTER COLUMN family_id SET NOT NULL`,
			`ALTER TABLE investimentos      ALTER COLUMN family_id SET NOT NULL`,
			`ALTER TABLE reserva_emergencia ALTER COLUMN family_id SET NOT NULL`,
			`ALTER TABLE emprestimos        ALTER COLUMN family_id SET NOT NULL`,
		},
	},
	{
		version: 8,
		name:    "seed_default_categories",
		stmts: []string{
			`INSERT INTO categorias (family_id, nome, grupo, cor)
			 SELECT f.id, cat.nome, cat.grupo, cat.cor
			 FROM families f
			 CROSS JOIN (VALUES
			     ('Básicas',     'basica', '#3498db'),
			     ('Cartão',      'cartao', '#f39c12'),
			     ('VR',          'vr',     '#27ae60'),
			     ('Alimentação', 'basica', '#e74c3c'),
			     ('Transporte',  'basica', '#9b59b6'),
			     ('Saúde',       'basica', '#e91e63'),
			     ('Moradia',     'basica', '#795548'),
			     ('Lazer',       'basica', '#00bcd4'),
			     ('Educação',    'basica', '#3f51b5'),
			     ('Serviços',    'basica', '#607d8b')
			 ) AS cat(nome, grupo, cor)
			 ON CONFLICT (family_id, nome) DO NOTHING`,
		},
	},
	{
		version: 9,
		name:    "create_parcelamentos_mes",
		stmts: []string{
			`CREATE TABLE IF NOT EXISTS parcelamentos_mes (
				parcelamento_id INTEGER NOT NULL REFERENCES parcelamentos(id) ON DELETE CASCADE,
				mes             DATE    NOT NULL,
				pago            BOOLEAN NOT NULL DEFAULT false,
				pago_em         TIMESTAMPTZ,
				PRIMARY KEY (parcelamento_id, mes)
			)`,
		},
	},
	{
		version: 10,
		name:    "reserva_em_tipo",
		stmts: []string{
			`ALTER TABLE reserva_emergencia ADD COLUMN IF NOT EXISTS tipo VARCHAR(10) NOT NULL DEFAULT 'deposito'`,
		},
	},
	{
		version: 11,
		name:    "transacoes_banco_e_pluggy_items",
		stmts: []string{
			`CREATE TABLE IF NOT EXISTS transacoes_banco (
				id              SERIAL PRIMARY KEY,
				family_id       INTEGER NOT NULL REFERENCES families(id),
				data            DATE NOT NULL,
				descricao       TEXT NOT NULL,
				valor           NUMERIC(14,2) NOT NULL,
				tipo            VARCHAR(10) NOT NULL,
				categoria       VARCHAR(100) NOT NULL DEFAULT '',
				origem          VARCHAR(20) NOT NULL,
				banco           VARCHAR(100) NOT NULL DEFAULT '',
				fit_id          VARCHAR(255) NOT NULL DEFAULT '',
				status          VARCHAR(20) NOT NULL DEFAULT 'pendente',
				despesa_id      INTEGER REFERENCES despesas_fixas(id),
				receita_id      INTEGER REFERENCES receitas(id),
				pluggy_item_id  VARCHAR(255) NOT NULL DEFAULT '',
				criado_em       TIMESTAMPTZ NOT NULL DEFAULT NOW()
			)`,
			`CREATE INDEX IF NOT EXISTS idx_transacoes_banco_family ON transacoes_banco(family_id)`,
			`CREATE INDEX IF NOT EXISTS idx_transacoes_banco_fit_id ON transacoes_banco(family_id, fit_id)`,
			`CREATE TABLE IF NOT EXISTS pluggy_items (
				id              SERIAL PRIMARY KEY,
				family_id       INTEGER NOT NULL REFERENCES families(id),
				item_id         VARCHAR(255) NOT NULL UNIQUE,
				connector_name  VARCHAR(255) NOT NULL DEFAULT '',
				status          VARCHAR(50) NOT NULL DEFAULT 'active',
				last_sync       TIMESTAMPTZ,
				criado_em       TIMESTAMPTZ NOT NULL DEFAULT NOW()
			)`,
			`CREATE INDEX IF NOT EXISTS idx_pluggy_items_family ON pluggy_items(family_id)`,
		},
	},
	{
		version: 12,
		name:    "integracoes_config",
		stmts: []string{
			`CREATE TABLE IF NOT EXISTS integracoes_config (
				id              SERIAL PRIMARY KEY,
				integracao      VARCHAR(50) NOT NULL UNIQUE,
				client_id       TEXT NOT NULL DEFAULT '',
				client_secret   TEXT NOT NULL DEFAULT '',
				service_url     TEXT NOT NULL DEFAULT '',
				ativo           BOOLEAN NOT NULL DEFAULT false,
				atualizado_em   TIMESTAMPTZ NOT NULL DEFAULT NOW()
			)`,
			`INSERT INTO integracoes_config (integracao, service_url)
			 VALUES ('pluggy', 'http://pluggy-service:8081')
			 ON CONFLICT (integracao) DO NOTHING`,
		},
	},
	{
		version: 13,
		name:    "financiamento",
		stmts: []string{
			`ALTER TABLE parcelamentos ADD COLUMN IF NOT EXISTS financiamento BOOLEAN NOT NULL DEFAULT false`,
			`ALTER TABLE parcelamentos ADD COLUMN IF NOT EXISTS valor_original NUMERIC(14,2) NOT NULL DEFAULT 0`,
			`ALTER TABLE parcelamentos ADD COLUMN IF NOT EXISTS taxa_juros NUMERIC(8,4) NOT NULL DEFAULT 0`,
			`ALTER TABLE parcelamentos ADD COLUMN IF NOT EXISTS total_economizado NUMERIC(14,2) NOT NULL DEFAULT 0`,
			`ALTER TABLE parcelamentos_mes ADD COLUMN IF NOT EXISTS valor_pago NUMERIC(14,2)`,
			`ALTER TABLE parcelamentos_mes ADD COLUMN IF NOT EXISTS antecipada BOOLEAN NOT NULL DEFAULT false`,
		},
	},
	{
		version: 14,
		name:    "password_resets",
		stmts: []string{
			`CREATE TABLE IF NOT EXISTS password_resets (
				id SERIAL PRIMARY KEY,
				user_id INTEGER NOT NULL REFERENCES users(id),
				token VARCHAR(64) NOT NULL UNIQUE,
				expira_em TIMESTAMP NOT NULL,
				usado BOOLEAN DEFAULT FALSE,
				criado_em TIMESTAMP DEFAULT NOW()
			)`,
		},
	},
	{
		version: 15,
		name:    "integracoes_extras",
		stmts: []string{
			`CREATE TABLE IF NOT EXISTS dados_economicos (
				id SERIAL PRIMARY KEY,
				tipo VARCHAR(20) NOT NULL,
				valor NUMERIC(18,6) NOT NULL,
				data DATE NOT NULL,
				fonte VARCHAR(30) NOT NULL,
				criado_em TIMESTAMP DEFAULT NOW(),
				UNIQUE(tipo, data, fonte)
			)`,
			`CREATE TABLE IF NOT EXISTS feriados (
				data DATE NOT NULL,
				nome VARCHAR(200) NOT NULL,
				tipo VARCHAR(30) DEFAULT 'national',
				PRIMARY KEY(data)
			)`,
			`CREATE TABLE IF NOT EXISTS integracoes_kv (
				family_id INTEGER NOT NULL,
				integracao VARCHAR(30) NOT NULL,
				chave VARCHAR(50) NOT NULL,
				valor TEXT NOT NULL DEFAULT '',
				PRIMARY KEY(family_id, integracao, chave)
			)`,
		},
	},
	{
		version: 16,
		name:    "dashboard_widgets",
		stmts: []string{
			`CREATE TABLE IF NOT EXISTS dashboard_widgets (
				user_id INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
				widget_key VARCHAR(40) NOT NULL,
				visivel BOOLEAN NOT NULL DEFAULT true,
				posicao INTEGER NOT NULL DEFAULT 0,
				PRIMARY KEY(user_id, widget_key)
			)`,
		},
	},
	{
		version: 17,
		name:    "user_notificacoes",
		stmts: []string{
			`CREATE TABLE IF NOT EXISTS user_notificacoes (
				user_id          INTEGER PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE,
				email_mensal     BOOLEAN NOT NULL DEFAULT false,
				telegram_mensal  BOOLEAN NOT NULL DEFAULT false,
				telegram_chat_id VARCHAR(50) NOT NULL DEFAULT '',
				dia_envio        INTEGER NOT NULL DEFAULT 1 CHECK (dia_envio BETWEEN 1 AND 31),
				mes_seguinte     BOOLEAN NOT NULL DEFAULT false,
				ultimo_envio     DATE
			)`,
		},
	},
	{
		version: 18,
		name:    "dashboard_widgets_coluna",
		stmts: []string{
			`ALTER TABLE dashboard_widgets ADD COLUMN IF NOT EXISTS coluna VARCHAR(10) NOT NULL DEFAULT ''`,
		},
	},
	{
		version: 19,
		name:    "orcamento_mensal_por_categoria",
		stmts: []string{
			// NULL = categoria sem orcamento.
			`ALTER TABLE categorias ADD COLUMN IF NOT EXISTS orcamento_mensal NUMERIC(14,2)
				CHECK (orcamento_mensal IS NULL OR orcamento_mensal > 0)`,
		},
	},
	{
		version: 20,
		name:    "alerta_de_orcamento",
		stmts: []string{
			`ALTER TABLE user_notificacoes ADD COLUMN IF NOT EXISTS alerta_orcamento_email BOOLEAN NOT NULL DEFAULT false`,
			`ALTER TABLE user_notificacoes ADD COLUMN IF NOT EXISTS alerta_orcamento_telegram BOOLEAN NOT NULL DEFAULT false`,
			// Um registro por usuario, categoria, mes e faixa: e o que impede o
			// mesmo alerta de sair de novo a cada hora.
			`CREATE TABLE IF NOT EXISTS alertas_de_orcamento_enviados (
				user_id      INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
				categoria_id INTEGER NOT NULL REFERENCES categorias(id) ON DELETE CASCADE,
				mes          DATE NOT NULL,
				faixa        VARCHAR(10) NOT NULL CHECK (faixa IN ('alerta', 'estourado')),
				enviado_em   TIMESTAMPTZ NOT NULL DEFAULT NOW(),
				PRIMARY KEY (user_id, categoria_id, mes, faixa)
			)`,
		},
	},
	{
		version: 21,
		name:    "regras_de_categorizacao",
		stmts: []string{
			// Categoria original que veio do banco (ex.: "Groceries" do Pluggy).
			// Fica guardada para as regras continuarem casando depois que a
			// categoria da transacao for trocada pela da familia.
			`ALTER TABLE transacoes_banco ADD COLUMN IF NOT EXISTS categoria_do_banco VARCHAR(255) NOT NULL DEFAULT ''`,
			`UPDATE transacoes_banco t SET categoria_do_banco = t.categoria
				WHERE t.categoria_do_banco = '' AND t.categoria <> ''
				  AND NOT EXISTS (SELECT 1 FROM categorias c WHERE c.family_id = t.family_id AND c.nome = t.categoria)`,
			`CREATE TABLE IF NOT EXISTS regras_de_categorizacao (
				id           SERIAL PRIMARY KEY,
				family_id    INTEGER NOT NULL REFERENCES families(id),
				campo        VARCHAR(20) NOT NULL CHECK (campo IN ('descricao', 'categoria_do_banco')),
				texto        VARCHAR(100) NOT NULL CHECK (texto <> ''),
				categoria_id INTEGER NOT NULL REFERENCES categorias(id) ON DELETE CASCADE,
				criado_em    TIMESTAMPTZ NOT NULL DEFAULT NOW(),
				UNIQUE (family_id, campo, texto)
			)`,
		},
	},
	{
		version: 22,
		name:    "metas_financeiras",
		stmts: []string{
			`CREATE TABLE IF NOT EXISTS metas (
				id         SERIAL PRIMARY KEY,
				family_id  INTEGER NOT NULL REFERENCES families(id),
				nome       VARCHAR(80) NOT NULL CHECK (nome <> ''),
				valor_alvo NUMERIC(14,2) NOT NULL CHECK (valor_alvo > 0),
				data_alvo  DATE NOT NULL,
				criado_em  TIMESTAMPTZ NOT NULL DEFAULT NOW()
			)`,
			`CREATE INDEX IF NOT EXISTS idx_metas_family ON metas(family_id)`,
			// Retirada entra como valor negativo; zero nao faz sentido.
			`CREATE TABLE IF NOT EXISTS metas_aportes (
				id        SERIAL PRIMARY KEY,
				meta_id   INTEGER NOT NULL REFERENCES metas(id) ON DELETE CASCADE,
				family_id INTEGER NOT NULL REFERENCES families(id),
				valor     NUMERIC(14,2) NOT NULL CHECK (valor <> 0),
				data      DATE NOT NULL DEFAULT CURRENT_DATE,
				criado_em TIMESTAMPTZ NOT NULL DEFAULT NOW()
			)`,
		},
	},
}

// RunMigrations executa as migrations pendentes registrando cada uma em schema_migrations.
func RunMigrations(db *sql.DB) error {
	_, err := db.Exec(`CREATE TABLE IF NOT EXISTS schema_migrations (
		version    INTEGER PRIMARY KEY,
		name       VARCHAR(200) NOT NULL,
		applied_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
	)`)
	if err != nil {
		return fmt.Errorf("criar schema_migrations: %w", err)
	}

	for _, m := range migrations {
		var applied bool
		db.QueryRow(`SELECT EXISTS(SELECT 1 FROM schema_migrations WHERE version=$1)`, m.version).Scan(&applied)
		if applied {
			continue
		}

		tx, err := db.Begin()
		if err != nil {
			return fmt.Errorf("migration %d: begin: %w", m.version, err)
		}
		for _, stmt := range m.stmts {
			if _, err := tx.Exec(stmt); err != nil {
				tx.Rollback()
				return fmt.Errorf("migration %d (%s): %w", m.version, m.name, err)
			}
		}
		if _, err := tx.Exec(`INSERT INTO schema_migrations (version, name) VALUES ($1, $2)`, m.version, m.name); err != nil {
			tx.Rollback()
			return fmt.Errorf("migration %d: registrar versão: %w", m.version, err)
		}
		if err := tx.Commit(); err != nil {
			return fmt.Errorf("migration %d: commit: %w", m.version, err)
		}
		log.Printf("[migrate] v%d (%s) aplicada", m.version, m.name)
	}
	return nil
}
