package handler

import (
	"database/sql"
	"encoding/json"
	"fincontrol/internal/auth"
	"fincontrol/internal/integrations"
	"fincontrol/internal/models"
	"fincontrol/internal/store"
	"net/http"
	"net/url"
	"strings"
	"time"
)

func buildIntegracoesList(db *sql.DB, fid int) []models.IntegracaoStatus {
	return []models.IntegracaoStatus{
		{
			Nome: "bcb", Label: "Banco Central (BCB)",
			Descricao: "Taxas Selic, CDI e IPCA atualizadas automaticamente.",
			Ativa:     store.GetIntegracaoAtiva(db, fid, "bcb"),
		},
		{
			Nome: "cotacoes", Label: "Cotações (AwesomeAPI)",
			Descricao: "Cotações do Dólar, Euro e Bitcoin em tempo real.",
			Ativa:     store.GetIntegracaoAtiva(db, fid, "cotacoes"),
		},
		{
			Nome: "telegram", Label: "Telegram Bot",
			Descricao: "Receba alertas de vencimento e resumo semanal no Telegram.",
			Ativa:     store.GetIntegracaoAtiva(db, fid, "telegram"),
			Campos: []models.IntegracaoCampo{
				{Nome: "bot_token", Label: "Bot Token", Tipo: "password", Segredo: true, Configurado: store.GetIntegracaoKV(db, fid, "telegram", "bot_token") != "", Placeholder: "Cole o token do BotFather"},
				{Nome: "chat_id", Label: "Chat ID", Tipo: "text", Valor: store.GetIntegracaoKV(db, fid, "telegram", "chat_id"), Placeholder: "Seu chat ID numérico"},
			},
		},
		{
			Nome: "brasilapi", Label: "Feriados (BrasilAPI)",
			Descricao: "Calendário de feriados nacionais do ano.",
			Ativa:     store.GetIntegracaoAtiva(db, fid, "brasilapi"),
		},
		{
			Nome: "sheets", Label: "Google Sheets",
			Descricao: "Atualiza uma planilha Google com contas, receitas, parcelamentos e investimentos (1x por dia).",
			Ativa:     store.GetIntegracaoAtiva(db, fid, "sheets"),
			Campos:    camposSheets(db, fid),
		},
	}
}

// camposSheets mostra o e-mail da service account (para compartilhar a planilha) sem expor a chave.
func camposSheets(db *sql.DB, fid int) []models.IntegracaoCampo {
	saJSON := store.GetIntegracaoKV(db, fid, "sheets", "service_account_json")
	ajudaSA := ""
	if sa, err := integrations.ParseServiceAccount(saJSON); err == nil {
		ajudaSA = "Compartilhe a planilha como Editor com: " + sa.ClientEmail
	}
	ajudaID := ""
	if ultima := store.GetIntegracaoKV(db, fid, "sheets", "ultima_exportacao"); ultima != "" {
		ajudaID = "Última atualização: " + ultima
	}
	return []models.IntegracaoCampo{
		{Nome: "spreadsheet_id", Label: "Link ou ID da planilha", Tipo: "text", Valor: store.GetIntegracaoKV(db, fid, "sheets", "spreadsheet_id"), Placeholder: "https://docs.google.com/spreadsheets/d/...", Ajuda: ajudaID},
		{Nome: "service_account_json", Label: "JSON da Service Account", Tipo: "textarea", Segredo: true, Configurado: saJSON != "", Placeholder: "Cole aqui o conteúdo do arquivo .json da chave", Ajuda: ajudaSA},
	}
}

// camposSecretos não são sobrescritos quando o formulário vem vazio.
var camposSecretos = map[string]bool{"bot_token": true, "service_account_json": true}

// valorParaSalvar decide se um campo do formulário de integração deve ser gravado.
func valorParaSalvar(campo string, vals []string) (string, bool) {
	if campo == "ativa" || len(vals) == 0 {
		return "", false
	}
	v := strings.TrimSpace(vals[0])
	if v == "" && camposSecretos[campo] {
		return "", false
	}
	return v, true
}

func HandleIntegracoes(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		u := auth.CurrentUser(r)
		sucesso := r.URL.Query().Get("ok")
		erro := r.URL.Query().Get("erro")

		var sucessoMsg, erroMsg string
		if sucesso != "" {
			sucessoMsg = "Integração atualizada com sucesso!"
		}
		if erro != "" {
			erroMsg = erro
		}

		page := models.IntegracoesPage{
			BasePage: models.BasePage{
				CurrentUser:    u,
				Active:         "integracoes",
				Title:          "Integrações",
				BlockedScreens: u.BlockedScreens,
			},
			Integracoes: buildIntegracoesList(db, u.FamilyID),
			Sucesso:     sucessoMsg,
			Erro:        erroMsg,
		}

		// Pluggy config (admin only)
		if u.Admin || u.FamilyAdmin {
			cfg, err := store.GetIntegracaoConfig(db, "pluggy")
			if err == nil {
				page.PluggyConfig = cfg
				page.PluggyContas, _ = store.GetPluggyItems(db, u.FamilyID)
				page.PluggyStatus = pluggyHealthCheck(cfg.ServiceURL)
			} else {
				page.PluggyStatus = "nao_configurado"
			}
		}

		render(w, "integracoes", page)
	}
}

func HandleIntegracaoSalvar(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		u := auth.CurrentUser(r)
		fid := u.FamilyID
		nome := r.PathValue("nome")
		r.ParseForm()

		ativa := r.FormValue("ativa") == "on"
		store.SetIntegracaoAtiva(db, fid, nome, ativa)

		erro := func(msg string) {
			http.Redirect(w, r, "/integracoes?erro="+url.QueryEscape(msg), http.StatusSeeOther)
		}
		switch nome {
		case "telegram":
			if tok := strings.TrimSpace(r.FormValue("bot_token")); tok != "" {
				if err := integrations.ValidarTokenTelegram(tok); err != nil {
					erro("Telegram: " + err.Error())
					return
				}
			}
		case "sheets":
			if js := strings.TrimSpace(r.FormValue("service_account_json")); js != "" {
				if _, err := integrations.ParseServiceAccount(js); err != nil {
					erro("Google Sheets: " + err.Error())
					return
				}
			}
			if r.Form.Has("spreadsheet_id") {
				r.Form.Set("spreadsheet_id", integrations.ExtrairIDPlanilha(r.FormValue("spreadsheet_id")))
			}
		}

		// Salvar campos extras (segredos em branco mantêm o valor salvo)
		for key, vals := range r.Form {
			if v, ok := valorParaSalvar(key, vals); ok {
				store.SetIntegracaoKV(db, fid, nome, key, v)
			}
		}

		// Ações especiais ao ativar
		if ativa {
			switch nome {
			case "bcb":
				go integrations.FetchBCB(db)
			case "cotacoes":
				go integrations.FetchCotacoes(db)
			case "brasilapi":
				go integrations.FetchFeriados(db, time.Now().Year())
			case "telegram":
				botToken := store.GetIntegracaoKV(db, fid, "telegram", "bot_token")
				chatID := store.GetIntegracaoKV(db, fid, "telegram", "chat_id")
				if botToken != "" && chatID != "" {
					if err := integrations.TestTelegram(botToken, chatID); err != nil {
						erro("Telegram: " + err.Error())
						return
					}
				}
			case "sheets":
				if err := integrations.ExportToSheet(db, fid); err != nil {
					erro("Google Sheets (configuração salva, mas a planilha não foi atualizada): " + err.Error())
					return
				}
			}
		}

		http.Redirect(w, r, "/integracoes?ok=1", http.StatusSeeOther)
	}
}

func HandleIntegracaoTestar(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		u := auth.CurrentUser(r)
		fid := u.FamilyID
		nome := r.PathValue("nome")

		w.Header().Set("Content-Type", "application/json")
		var err error

		switch nome {
		case "bcb":
			err = integrations.FetchBCB(db)
		case "cotacoes":
			err = integrations.FetchCotacoes(db)
		case "brasilapi":
			err = integrations.FetchFeriados(db, time.Now().Year())
		case "telegram":
			botToken := store.GetIntegracaoKV(db, fid, "telegram", "bot_token")
			chatID := store.GetIntegracaoKV(db, fid, "telegram", "chat_id")
			err = integrations.TestTelegram(botToken, chatID)
		case "sheets":
			err = integrations.ExportToSheet(db, fid)
		}

		if err != nil {
			json.NewEncoder(w).Encode(map[string]string{"status": "erro", "mensagem": err.Error()})
		} else {
			json.NewEncoder(w).Encode(map[string]string{"status": "ok", "mensagem": "Teste realizado com sucesso!"})
		}
	}
}
