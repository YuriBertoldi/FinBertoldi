package integrations

import (
	"database/sql"
	"log"
	"time"

	"fincontrol/internal/store"
)

// StartScheduler inicia as rotinas periódicas das integrações
func StartScheduler(db *sql.DB) {
	// Buscar todas as famílias
	go func() {
		// Aguardar startup
		time.Sleep(10 * time.Second)

		// Executar imediatamente na primeira vez
		runGlobalIntegrations(db)
		runFamilyIntegrations(db)

		// Ticker para dados globais (BCB, cotações, feriados) - a cada 6h
		globalTicker := time.NewTicker(6 * time.Hour)
		// Ticker para integrações por família (telegram) - a cada 24h
		familyTicker := time.NewTicker(24 * time.Hour)

		for {
			select {
			case <-globalTicker.C:
				runGlobalIntegrations(db)
			case <-familyTicker.C:
				runFamilyIntegrations(db)
			}
		}
	}()
}

func runGlobalIntegrations(db *sql.DB) {
	log.Println("[integracoes] executando integrações globais...")

	if err := FetchBCB(db); err != nil {
		log.Printf("[integracoes] erro BCB: %v", err)
	}

	if err := FetchCotacoes(db); err != nil {
		log.Printf("[integracoes] erro cotações: %v", err)
	}

	ano := time.Now().Year()
	if err := FetchFeriados(db, ano); err != nil {
		log.Printf("[integracoes] erro feriados: %v", err)
	}
}

func runFamilyIntegrations(db *sql.DB) {
	exportarPlanilhas(db)

	// Buscar famílias com telegram ativo
	rows, err := db.Query(`SELECT DISTINCT family_id FROM integracoes_kv WHERE integracao='telegram' AND chave='ativa' AND valor='true'`)
	if err != nil {
		return
	}
	defer rows.Close()

	for rows.Next() {
		var fid int
		rows.Scan(&fid)

		if store.GetIntegracaoAtiva(db, fid, "telegram") {
			if err := SendAlertaVencimento(db, fid); err != nil {
				log.Printf("[telegram] erro alerta fid=%d: %v", fid, err)
			}
		}
	}
}

// exportarPlanilhas atualiza o Google Sheets das famílias com a integração ativa (1x por dia).
func exportarPlanilhas(db *sql.DB) {
	rows, err := db.Query(`SELECT DISTINCT family_id FROM integracoes_kv WHERE integracao='sheets' AND chave='ativa' AND valor='true'`)
	if err != nil {
		return
	}
	var fids []int
	for rows.Next() {
		var fid int
		if rows.Scan(&fid) == nil {
			fids = append(fids, fid)
		}
	}
	rows.Close()
	for _, fid := range fids {
		if err := ExportToSheet(db, fid); err != nil {
			log.Printf("[sheets] erro fid=%d: %v", fid, err)
		}
	}
}
