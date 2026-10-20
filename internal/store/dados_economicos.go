package store

import (
	"database/sql"
	"fincontrol/internal/models"
	"time"
)

func UpsertDadoEconomico(db *sql.DB, tipo string, valor float64, data time.Time, fonte string) error {
	_, err := db.Exec(`INSERT INTO dados_economicos (tipo, valor, data, fonte)
		VALUES ($1,$2,$3,$4) ON CONFLICT (tipo, data, fonte) DO UPDATE SET valor=$2`,
		tipo, valor, data, fonte)
	return err
}

func GetUltimoDadoEconomico(db *sql.DB, tipo string) (float64, error) {
	var v float64
	err := db.QueryRow(`SELECT valor FROM dados_economicos WHERE tipo=$1 ORDER BY data DESC LIMIT 1`, tipo).Scan(&v)
	return v, err
}

func GetDadosEconomicos(db *sql.DB) models.DashboardIntegracoes {
	var d models.DashboardIntegracoes
	tipos := []struct {
		nome string
		dest **float64
	}{
		{"selic", &d.Selic}, {"cdi", &d.CDI}, {"ipca", &d.IPCA},
		{"usd", &d.USD}, {"eur", &d.EUR}, {"btc", &d.BTC},
	}
	for _, t := range tipos {
		v, err := GetUltimoDadoEconomico(db, t.nome)
		if err == nil {
			val := v
			*t.dest = &val
		}
	}
	return d
}
