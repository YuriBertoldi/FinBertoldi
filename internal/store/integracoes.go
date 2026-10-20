package store

import (
	"database/sql"
	"fincontrol/internal/models"
)

func SetIntegracaoKV(db *sql.DB, fid int, integracao, chave, valor string) error {
	_, err := db.Exec(`INSERT INTO integracoes_kv (family_id, integracao, chave, valor)
		VALUES ($1,$2,$3,$4) ON CONFLICT (family_id, integracao, chave) DO UPDATE SET valor=$4`,
		fid, integracao, chave, valor)
	return err
}

func GetIntegracaoKV(db *sql.DB, fid int, integracao, chave string) string {
	var val string
	db.QueryRow(`SELECT valor FROM integracoes_kv WHERE family_id=$1 AND integracao=$2 AND chave=$3`,
		fid, integracao, chave).Scan(&val)
	return val
}

func GetIntegracaoAtiva(db *sql.DB, fid int, integracao string) bool {
	return GetIntegracaoKV(db, fid, integracao, "ativa") == "true"
}

func SetIntegracaoAtiva(db *sql.DB, fid int, integracao string, ativa bool) error {
	v := "false"
	if ativa {
		v = "true"
	}
	return SetIntegracaoKV(db, fid, integracao, "ativa", v)
}

func GetIntegracaoConfig(db *sql.DB, integracao string) (*models.IntegracaoConfig, error) {
	var c models.IntegracaoConfig
	err := db.QueryRow(`SELECT id, integracao, client_id, client_secret, service_url, ativo
		FROM integracoes_config WHERE integracao=$1`, integracao).
		Scan(&c.ID, &c.Integracao, &c.ClientID, &c.ClientSecret, &c.ServiceURL, &c.Ativo)
	if err != nil {
		return nil, err
	}
	return &c, nil
}

func SaveIntegracaoConfig(db *sql.DB, c models.IntegracaoConfig) error {
	_, err := db.Exec(`INSERT INTO integracoes_config (integracao, client_id, client_secret, service_url, ativo, atualizado_em)
		VALUES ($1, $2, $3, $4, $5, NOW())
		ON CONFLICT (integracao) DO UPDATE SET
			client_id=EXCLUDED.client_id,
			client_secret=EXCLUDED.client_secret,
			service_url=EXCLUDED.service_url,
			ativo=EXCLUDED.ativo,
			atualizado_em=NOW()`,
		c.Integracao, c.ClientID, c.ClientSecret, c.ServiceURL, c.Ativo)
	return err
}
