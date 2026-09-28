package handler

import (
	"bytes"
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"html/template"
	"io"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"fincontrol/internal/auth"
	"fincontrol/internal/integrations"
	"fincontrol/internal/models"
	"fincontrol/internal/store"
	"golang.org/x/crypto/bcrypt"
)

// --- Template rendering ---

var tmplPages = map[string]*template.Template{}

func buildFuncMap() template.FuncMap {
	return template.FuncMap{
		"brl": func(v float64) template.HTML {
			return template.HTML(`<span class="money">` + formatBRL(v) + `</span>`)
		},
		"date":       func(t time.Time) string { return t.Format("02/01/2006") },
		"dateInput":  func(t time.Time) string { return t.Format("2006-01-02") },
		"neg":        func(v float64) bool { return v < 0 },
		"abs":        func(v float64) float64 { if v < 0 { return -v }; return v },
		"progress":   func(atual, total int) string { return fmt.Sprintf("%dx%d", atual, total) },
		"sub":        func(a, b float64) float64 { return a - b },
		"pct": func(v, total float64) string {
			if total == 0 { return "0%" }
			return fmt.Sprintf("%.0f%%", (v/total)*100)
		},
		"mesDisplay": mesDisplay,
		"seq": func(n int) []int {
			s := make([]int, n)
			for i := range s { s[i] = i + 1 }
			return s
		},
		"json": func(v any) template.JS {
			b, _ := json.Marshal(v)
			return template.JS(b)
		},
		"deref": func(p *float64) float64 {
			if p == nil { return 0 }
			return *p
		},
		"contains": func(slice []string, s string) bool {
			for _, v := range slice {
				if v == s { return true }
			}
			return false
		},
	}
}

// InitTemplates carrega e compila todos os templates.
func InitTemplates() {
	pages := []string{"dashboard", "despesas", "pagamentos", "receitas", "investimentos", "planejamento", "emprestimos", "usuarios", "familias", "cadastros", "minha-familia", "importexport", "transacoes", "integracoes"}
	for _, page := range pages {
		t := template.New("").Funcs(buildFuncMap())
		template.Must(t.ParseFiles("templates/base.html", "templates/"+page+".html"))
		tmplPages[page] = t
	}
	lt := template.Must(template.New("login").Funcs(buildFuncMap()).ParseFiles("templates/login.html"))
	tmplPages["login"] = lt
	fp := template.Must(template.New("forgot_password").Funcs(buildFuncMap()).ParseFiles("templates/forgot-password.html"))
	tmplPages["forgot_password"] = fp
	rp := template.Must(template.New("reset_password").Funcs(buildFuncMap()).ParseFiles("templates/reset-password.html"))
	tmplPages["reset_password"] = rp
	InitReportTemplates()
}

func render(w http.ResponseWriter, page string, data any) {
	t, ok := tmplPages[page]
	if !ok {
		http.Error(w, "template não encontrado: "+page, http.StatusInternalServerError)
		return
	}
	if err := t.ExecuteTemplate(w, "base", data); err != nil {
		log.Printf("template %s error: %v", page, err)
		http.Error(w, "Erro interno", http.StatusInternalServerError)
	}
}

func renderLogin(w http.ResponseWriter, data any) {
	t, ok := tmplPages["login"]
	if !ok {
		http.Error(w, "template login não encontrado", http.StatusInternalServerError)
		return
	}
	if err := t.ExecuteTemplate(w, "login", data); err != nil {
		log.Printf("login template error: %v", err)
	}
}

func renderPartial(w http.ResponseWriter, page, name string, data any) {
	t, ok := tmplPages[page]
	if !ok {
		http.Error(w, "template não encontrado: "+page, http.StatusInternalServerError)
		return
	}
	if err := t.ExecuteTemplate(w, name, data); err != nil {
		log.Printf("partial %s/%s error: %v", page, name, err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

// --- Helpers ---

func bp(db *sql.DB, r *http.Request, active, title string) models.BasePage {
	u := auth.CurrentUser(r)
	page := models.BasePage{CurrentUser: u, Active: active, Title: title}
	if u != nil && !u.Admin && !u.FamilyAdmin {
		page.BlockedScreens, _ = store.GetBlockedScreens(db, u.ID)
	}
	return page
}

func mesFromRequest(r *http.Request) (time.Time, string) {
	mesStr := r.URL.Query().Get("mes")
	if mesStr == "" {
		mesStr = time.Now().Format("2006-01")
	}
	t, err := time.Parse("2006-01", mesStr)
	if err != nil {
		t = time.Now()
		mesStr = t.Format("2006-01")
	}
	return time.Date(t.Year(), t.Month(), 1, 0, 0, 0, 0, time.UTC), mesStr
}

func mesDisplay(s string) string {
	t, err := time.Parse("2006-01", s)
	if err != nil {
		return s
	}
	meses := []string{"", "Janeiro", "Fevereiro", "Março", "Abril", "Maio", "Junho",
		"Julho", "Agosto", "Setembro", "Outubro", "Novembro", "Dezembro"}
	return fmt.Sprintf("%s/%d", meses[t.Month()], t.Year())
}

func formatBRL(v float64) string {
	neg := v < 0
	if neg { v = -v }
	cents := int64(v*100+0.5) % 100
	whole := int64(v)
	s := strconv.FormatInt(whole, 10)
	if len(s) > 3 {
		var b strings.Builder
		for i, c := range s {
			if i > 0 && (len(s)-i)%3 == 0 { b.WriteRune('.') }
			b.WriteRune(c)
		}
		s = b.String()
	}
	result := fmt.Sprintf("R$ %s,%02d", s, cents)
	if neg { result = "-" + result }
	return result
}

func parseInt(s string) int {
	n, _ := strconv.Atoi(strings.TrimSpace(s))
	return n
}

func parseFloat(s string) float64 {
	s = strings.TrimSpace(s)
	hasDot := strings.Contains(s, ".")
	hasComma := strings.Contains(s, ",")
	if hasDot && hasComma {
		// Formato BR: 1.234,56 — ponto é milhar, vírgula é decimal
		s = strings.ReplaceAll(s, ".", "")
		s = strings.ReplaceAll(s, ",", ".")
	} else if hasComma {
		// Só vírgula: 223,05 → 223.05
		s = strings.ReplaceAll(s, ",", ".")
	}
	// Só ponto ou sem separador: já é float padrão (223.05)
	f, _ := strconv.ParseFloat(s, 64)
	return f
}

func parseDate(s string) time.Time {
	t, err := time.Parse("2006-01-02", strings.TrimSpace(s))
	if err != nil { return time.Now() }
	return t
}

// TelasDispo lista as telas que podem ter acesso controlado.
var TelasDispo = []models.ScreenInfo{
	{"despesas", "💸 Despesas"},
	{"receitas", "💵 Receitas"},
	{"investimentos", "📈 Investimentos"},
	{"planejamento", "🎯 Planejamento"},
	{"emprestimos", "🤝 Empréstimos"},
	{"cadastros", "🗂️ Cadastros"},
}

// --- Login / Logout ---

func HandleLogin(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if u := auth.AuthUser(db, r); u != nil && u.Ativo {
			http.Redirect(w, r, "/", http.StatusSeeOther)
			return
		}
		if r.Method == http.MethodGet {
			erro := ""
			if r.URL.Query().Get("erro") != "" {
				erro = "Email ou senha incorretos."
			}
			if r.URL.Query().Get("reset") != "" {
				erro = "Senha alterada com sucesso! Faca login."
			}
			renderLogin(w, models.LoginPage{Title: "Login — FinBertoldi", Erro: erro})
			return
		}
		r.ParseForm()
		email := strings.TrimSpace(r.FormValue("email"))
		senha := r.FormValue("senha")

		user, hash, err := store.GetUserByEmail(db, email)
		if err != nil || !user.Ativo {
			http.Redirect(w, r, "/login?erro=1", http.StatusSeeOther)
			return
		}
		if bcrypt.CompareHashAndPassword([]byte(hash), []byte(senha)) != nil {
			http.Redirect(w, r, "/login?erro=1", http.StatusSeeOther)
			return
		}

		token := auth.GenerateToken()
		expiry := time.Now().Add(7 * 24 * time.Hour)
		if err := store.CreateSession(db, user.ID, token, expiry); err != nil {
			log.Printf("create session: %v", err)
		}
		auth.SetSessionCookie(w, token, expiry)
		http.Redirect(w, r, "/", http.StatusSeeOther)
	}
}

func HandleLogout(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if c, err := r.Cookie("session"); err == nil {
			store.DeleteSession(db, c.Value)
		}
		auth.ClearSessionCookie(w)
		http.Redirect(w, r, "/login", http.StatusSeeOther)
	}
}

// --- Recuperação de Senha ---

func renderStandalone(w http.ResponseWriter, name string, data any) {
	t, ok := tmplPages[name]
	if !ok {
		http.Error(w, "template não encontrado: "+name, http.StatusInternalServerError)
		return
	}
	if err := t.ExecuteTemplate(w, name, data); err != nil {
		log.Printf("%s template error: %v", name, err)
	}
}

func HandleForgotPassword(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet {
			msg := ""
			if r.URL.Query().Get("ok") != "" {
				msg = "Se o email estiver cadastrado, enviaremos um link de recuperacao."
			}
			renderStandalone(w, "forgot_password", map[string]string{"Msg": msg})
			return
		}
		r.ParseForm()
		email := strings.TrimSpace(r.FormValue("email"))

		// Sempre redireciona com mensagem genérica (segurança)
		userID, err := store.GetUserIDByEmail(db, email)
		if err == nil && userID > 0 {
			tokenBytes := make([]byte, 32)
			rand.Read(tokenBytes)
			token := hex.EncodeToString(tokenBytes)
			expiry := time.Now().Add(1 * time.Hour)

			if err := store.CreatePasswordReset(db, userID, token, expiry); err != nil {
				log.Printf("[password-reset] erro ao criar token: %v", err)
			} else {
				baseURL := os.Getenv("BASE_URL")
				if baseURL == "" {
					baseURL = "http://localhost:8080"
				}
				if err := auth.SendResetEmail(email, token, baseURL); err != nil {
					log.Printf("[password-reset] erro ao enviar email: %v", err)
				}
			}
		}
		http.Redirect(w, r, "/forgot-password?ok=1", http.StatusSeeOther)
	}
}

func HandleResetPassword(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		token := r.URL.Query().Get("token")
		if r.Method == http.MethodGet {
			if token == "" {
				renderStandalone(w, "reset_password", map[string]string{"Erro": "Link invalido."})
				return
			}
			_, err := store.GetPasswordReset(db, token)
			if err != nil {
				renderStandalone(w, "reset_password", map[string]string{"Erro": "Link expirado ou invalido."})
				return
			}
			renderStandalone(w, "reset_password", map[string]string{"Token": token})
			return
		}
		r.ParseForm()
		token = r.FormValue("token")
		senha := r.FormValue("senha")
		confirmar := r.FormValue("confirmar")

		if senha != confirmar {
			renderStandalone(w, "reset_password", map[string]string{"Token": token, "Erro": "As senhas nao coincidem."})
			return
		}
		if len(senha) < 6 {
			renderStandalone(w, "reset_password", map[string]string{"Token": token, "Erro": "A senha deve ter pelo menos 6 caracteres."})
			return
		}

		userID, err := store.GetPasswordReset(db, token)
		if err != nil {
			renderStandalone(w, "reset_password", map[string]string{"Erro": "Link expirado ou invalido."})
			return
		}

		hashed, err := bcrypt.GenerateFromPassword([]byte(senha), bcrypt.DefaultCost)
		if err != nil {
			renderStandalone(w, "reset_password", map[string]string{"Erro": "Erro interno."})
			return
		}

		store.UpdateUserPassword(db, userID, string(hashed))
		store.UsePasswordReset(db, token)

		http.Redirect(w, r, "/login?reset=1", http.StatusSeeOther)
	}
}

// --- Integrações ---

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
				{Nome: "bot_token", Label: "Bot Token", Tipo: "password", Valor: store.GetIntegracaoKV(db, fid, "telegram", "bot_token"), Placeholder: "Cole o token do BotFather"},
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
			Descricao: "Exporte dados automaticamente para uma planilha Google.",
			Ativa:     store.GetIntegracaoAtiva(db, fid, "sheets"),
			Campos: []models.IntegracaoCampo{
				{Nome: "spreadsheet_id", Label: "ID da Planilha", Tipo: "text", Valor: store.GetIntegracaoKV(db, fid, "sheets", "spreadsheet_id"), Placeholder: "ID da planilha no Google Sheets"},
				{Nome: "service_account_json", Label: "Service Account JSON", Tipo: "textarea", Valor: store.GetIntegracaoKV(db, fid, "sheets", "service_account_json"), Placeholder: "Cole o JSON da service account"},
			},
		},
	}
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

		render(w, "integracoes", models.IntegracoesPage{
			BasePage: models.BasePage{
				CurrentUser:    u,
				Active:         "integracoes",
				Title:          "Integrações",
				BlockedScreens: u.BlockedScreens,
			},
			Integracoes: buildIntegracoesList(db, u.FamilyID),
			Sucesso:     sucessoMsg,
			Erro:        erroMsg,
		})
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

		// Salvar campos extras
		for key, vals := range r.Form {
			if key == "ativa" {
				continue
			}
			if len(vals) > 0 {
				store.SetIntegracaoKV(db, fid, nome, key, vals[0])
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
				botToken := r.FormValue("bot_token")
				chatID := r.FormValue("chat_id")
				if botToken != "" && chatID != "" {
					if err := integrations.TestTelegram(botToken, chatID); err != nil {
						http.Redirect(w, r, "/integracoes?erro=Telegram: "+err.Error(), http.StatusSeeOther)
						return
					}
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

// --- Dashboard ---

func HandleDashboard(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		fid := auth.CurrentUser(r).FamilyID
		mes, mesStr := mesFromRequest(r)
		prevMes := mes.AddDate(0, -1, 0).Format("2006-01")
		nextMes := mes.AddDate(0, 1, 0).Format("2006-01")

		if err := store.EnsureDespesasMes(db, fid, mes); err != nil {
			log.Printf("ensure despesas mes: %v", err)
		}
		if err := store.AutoFinalizarParcelamentos(db, fid); err != nil {
			log.Printf("auto finalizar parcelamentos: %v", err)
		}

		basicas, cartao, vr, err := store.GetDespesasMes(db, fid, mes, mesStr)
		if err != nil {
			log.Printf("get despesas mes: %v", err)
		}
		parcelamentos, _ := store.GetParcelamentosVigentesNoMes(db, fid, mes)
		totalReceitas, _ := store.GetReceitasMes(db, fid, mes)
		_, totalInvestido, _ := store.GetInvestimentosTotais(db, fid)
		investidoNoMes := store.GetInvestidoNoMes(db, fid, mes)
		reservaEM, _ := store.GetReservaEM(db, fid)
		historico, _ := store.GetHistoricoMeses(db, fid, mes, 6)

		// Reserva de emergência conta como investimento no total
		resumo := calcResumo(basicas, cartao, vr, parcelamentos, totalReceitas, totalInvestido+reservaEM, investidoNoMes)

		render(w, "dashboard", models.DashboardData{
			BasePage:        bp(db, r, "dashboard", "Dashboard"),
			DashboardResumo: resumo,
			Mes:             mesStr,
			MesDisplay:      mesDisplay(mesStr),
			MesPrev:         prevMes,
			MesNext:         nextMes,
			ReservaEM:       reservaEM,
			DespesasBasicas: basicas,
			DespesasCartao:  cartao,
			DespesasVR:      vr,
			Parcelamentos:   parcelamentos,
			Historico:       historico,
			Integracoes:     store.GetDadosEconomicos(db),
		})
	}
}

// calcResumo agrega os totais do mês para os cards de resumo.
func calcResumo(basicas, cartao, vr []models.DespesaMes, parcs []models.Parcelamento, totalReceitas, totalInvestido, investidoNoMes float64) models.DashboardResumo {
	var totalDespesas, totalPago, totalPendente float64
	for _, d := range append(append(basicas, cartao...), vr...) {
		totalDespesas += d.Valor
		if d.Pago {
			totalPago += d.Valor
		} else {
			totalPendente += d.Valor
		}
	}
	for _, p := range parcs {
		totalDespesas += p.ValorParcela
		if p.Pago {
			totalPago += p.ValorParcela
		} else {
			totalPendente += p.ValorParcela
		}
	}
	return models.DashboardResumo{
		TotalReceitas:  totalReceitas,
		TotalDespesas:  totalDespesas,
		Sobra:          totalReceitas - totalDespesas,
		Caixa:          totalReceitas - totalPago - investidoNoMes,
		TotalInvestido: totalInvestido,
		TotalPago:      totalPago,
		TotalPendente:  totalPendente,
	}
}

// --- Toggle pago (HTMX) ---

func HandleToggleParcelamentoPago(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		fid := auth.CurrentUser(r).FamilyID
		id := parseInt(r.PathValue("id"))
		mes, mesStr := mesFromRequest(r)

		pago, err := store.ToggleParcelamentoMesPago(db, fid, id, mes)
		if err != nil {
			http.Error(w, err.Error(), 500)
			return
		}

		var p models.Parcelamento
		db.QueryRow(`SELECT id, descricao, cartao, valor_parcela FROM parcelamentos WHERE id=$1 AND family_id=$2`, id, fid).
			Scan(&p.ID, &p.Descricao, &p.Cartao, &p.ValorParcela)
		p.Mes = mesStr
		p.Pago = pago

		basicas, cartao, vr, _ := store.GetDespesasMes(db, fid, mes, mesStr)
		parcs, _ := store.GetParcelamentosVigentesNoMes(db, fid, mes)
		totalReceitas, _ := store.GetReceitasMes(db, fid, mes)
		_, totalInvestido, _ := store.GetInvestimentosTotais(db, fid)
		investidoNoMes := store.GetInvestidoNoMes(db, fid, mes)
		reservaEMp, _ := store.GetReservaEM(db, fid)
		resumo := calcResumo(basicas, cartao, vr, parcs, totalReceitas, totalInvestido+reservaEMp, investidoNoMes)

		t := tmplPages["dashboard"]
		var buf bytes.Buffer
		t.ExecuteTemplate(&buf, "toggle-parc-btn", p)
		t.ExecuteTemplate(&buf, "dash-resumo-oob", resumo)
		w.Write(buf.Bytes())
	}
}

func HandleToggleDespesaPago(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		fid := auth.CurrentUser(r).FamilyID
		id := parseInt(r.PathValue("id"))
		mes, mesStr := mesFromRequest(r)

		pago, err := store.ToggleDespesaMesPago(db, fid, id, mes)
		if err != nil {
			http.Error(w, err.Error(), 500)
			return
		}

		var d models.DespesaMes
		db.QueryRow(`SELECT id, nome, valor, categoria FROM despesas_fixas WHERE id=$1 AND family_id=$2`, id, fid).
			Scan(&d.ID, &d.Nome, &d.Valor, &d.Categoria)
		d.Mes = mesStr
		d.Pago = pago

		// Recalcula totais para o OOB swap dos cards
		basicas, cartao, vr, _ := store.GetDespesasMes(db, fid, mes, mesStr)
		parcs, _ := store.GetParcelamentosVigentesNoMes(db, fid, mes)
		totalReceitas, _ := store.GetReceitasMes(db, fid, mes)
		_, totalInvestido, _ := store.GetInvestimentosTotais(db, fid)
		investidoNoMes2 := store.GetInvestidoNoMes(db, fid, mes)
		reservaEMd, _ := store.GetReservaEM(db, fid)
		resumo := calcResumo(basicas, cartao, vr, parcs, totalReceitas, totalInvestido+reservaEMd, investidoNoMes2)

		t := tmplPages["dashboard"]
		var buf bytes.Buffer
		t.ExecuteTemplate(&buf, "toggle-btn", d)
		t.ExecuteTemplate(&buf, "dash-resumo-oob", resumo)
		w.Write(buf.Bytes())
	}
}

// --- Despesas ---

func HandleDespesas(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		fid := auth.CurrentUser(r).FamilyID
		tab := r.URL.Query().Get("tab")
		if tab == "" { tab = "fixas" }
		mes, mesStr := mesFromRequest(r)
		despesas, _ := store.GetDespesasFixas(db, fid)
		parcelamentos, _ := store.GetParcelamentos(db, fid, false)
		categorias, _ := store.GetCategorias(db, fid)
		cartoes, _ := store.GetCartoes(db, fid)
		store.EnsureDespesasMes(db, fid, mes)
		basicas, cartao, vr, _ := store.GetDespesasMes(db, fid, mes, mesStr)
		parcelamentosMes, _ := store.GetParcelamentosVigentesNoMes(db, fid, mes)

		render(w, "despesas", models.DespesasPage{
			BasePage:         bp(db, r, "despesas", "Despesas"),
			TabAtivo:         tab,
			Despesas:         despesas,
			Parcelamentos:    parcelamentos,
			Categorias:       categorias,
			Cartoes:          cartoes,
			MesStr:           mesStr,
			MesDisplay:       mesDisplay(mesStr),
			MesPrev:          mes.AddDate(0, -1, 0).Format("2006-01"),
			MesNext:          mes.AddDate(0, 1, 0).Format("2006-01"),
			DespesasBasicas:  basicas,
			DespesasCartao:   cartao,
			DespesasVR:       vr,
			ParcelamentosMes: parcelamentosMes,
		})
	}
}

func HandleCreateDespesa(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		r.ParseForm()
		store.CreateDespesa(db, auth.CurrentUser(r).FamilyID, r.FormValue("nome"), parseFloat(r.FormValue("valor")), r.FormValue("categoria"))
		http.Redirect(w, r, "/despesas", http.StatusSeeOther)
	}
}

func HandleUpdateDespesa(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		r.ParseForm()
		store.UpdateDespesa(db, auth.CurrentUser(r).FamilyID, parseInt(r.PathValue("id")),
			r.FormValue("nome"), parseFloat(r.FormValue("valor")), r.FormValue("categoria"))
		http.Redirect(w, r, "/despesas", http.StatusSeeOther)
	}
}

func HandleDeleteDespesa(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		store.DeleteDespesa(db, auth.CurrentUser(r).FamilyID, parseInt(r.PathValue("id")))
		http.Redirect(w, r, "/despesas", http.StatusSeeOther)
	}
}

func HandleToggleDespesaAtiva(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		store.ToggleDespesaAtiva(db, auth.CurrentUser(r).FamilyID, parseInt(r.PathValue("id")))
		http.Redirect(w, r, "/despesas", http.StatusSeeOther)
	}
}

// --- Parcelamentos ---

func HandleParcelamentos(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "/despesas?tab=parcelamentos", http.StatusSeeOther)
	}
}

func HandleCreateParcelamento(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		r.ParseForm()
		financiamento := r.FormValue("financiamento") == "on"
		var valorOriginal, taxaJuros float64
		if financiamento {
			valorOriginal = parseFloat(r.FormValue("valor_original"))
			taxaJuros = parseFloat(r.FormValue("taxa_juros"))
		}
		store.CreateParcelamento(db, auth.CurrentUser(r).FamilyID,
			r.FormValue("descricao"), r.FormValue("cartao"),
			parseFloat(r.FormValue("valor_parcela")),
			parseInt(r.FormValue("parcela_atual")),
			parseInt(r.FormValue("total_parcelas")),
			parseDate(r.FormValue("data_inicio")),
			financiamento, valorOriginal, taxaJuros)
		http.Redirect(w, r, "/despesas?tab=parcelamentos", http.StatusSeeOther)
	}
}

func HandleUpdateParcelamento(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		r.ParseForm()
		store.UpdateParcelamento(db, auth.CurrentUser(r).FamilyID, parseInt(r.PathValue("id")),
			parseInt(r.FormValue("parcela_atual")), r.FormValue("ativo") == "true")
		http.Redirect(w, r, "/despesas?tab=parcelamentos", http.StatusSeeOther)
	}
}

func HandleDeleteParcelamento(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		store.DeleteParcelamento(db, auth.CurrentUser(r).FamilyID, parseInt(r.PathValue("id")))
		http.Redirect(w, r, "/despesas?tab=parcelamentos", http.StatusSeeOther)
	}
}

func HandleAnteciparPreview(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		fid := auth.CurrentUser(r).FamilyID
		id := parseInt(r.PathValue("id"))
		previews, err := store.GetAntecipacoesPreview(db, fid, id)
		if err != nil {
			http.Error(w, err.Error(), 400)
			return
		}
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		if len(previews) == 0 {
			fmt.Fprint(w, `<p class="muted">Nenhuma parcela disponível para antecipação.</p>`)
			return
		}
		fmt.Fprint(w, `<div class="table-responsive"><table><thead><tr><th>Parcela</th><th>Valor Original</th><th>Desconto</th><th>Valor c/ Desconto</th></tr></thead><tbody>`)
		var totalDesc, totalPagar float64
		for _, p := range previews {
			totalDesc += p.Desconto
			totalPagar += p.ValorComDesconto
			fmt.Fprintf(w, `<tr><td>%dª</td><td>%s</td><td class="text-success">- %s</td><td><strong>%s</strong></td></tr>`,
				p.ParcelaNum, formatBRL(p.ValorOriginal), formatBRL(p.Desconto), formatBRL(p.ValorComDesconto))
		}
		fmt.Fprint(w, `</tbody></table></div>`)
		fmt.Fprintf(w, `<div class="antecipar-resumo"><p>Total a pagar: <strong>%s</strong></p><p class="text-success">Economia total: <strong>%s</strong></p></div>`,
			formatBRL(totalPagar), formatBRL(totalDesc))
	}
}

func HandleAnteciparParcela(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		r.ParseForm()
		fid := auth.CurrentUser(r).FamilyID
		id := parseInt(r.PathValue("id"))
		qtd := parseInt(r.FormValue("qtd_parcelas"))
		if qtd <= 0 {
			qtd = 1
		}
		economia, err := store.AnteciparParcelas(db, fid, id, qtd)
		if err != nil {
			log.Printf("antecipar parcela: %v", err)
			http.Redirect(w, r, "/despesas?tab=parcelamentos", http.StatusSeeOther)
			return
		}
		log.Printf("antecipação: %d parcelas, economia R$ %.2f", qtd, economia)
		http.Redirect(w, r, "/despesas?tab=parcelamentos", http.StatusSeeOther)
	}
}

// --- Receitas ---

func HandleReceitas(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		list, _ := store.GetReceitas(db, auth.CurrentUser(r).FamilyID)
		render(w, "receitas", models.ReceitasPage{BasePage: bp(db, r, "receitas", "Receitas"), Receitas: list})
	}
}

func HandleCreateReceita(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		r.ParseForm()
		store.CreateReceita(db, auth.CurrentUser(r).FamilyID, r.FormValue("descricao"), parseFloat(r.FormValue("valor")),
			parseDate(r.FormValue("data")), r.FormValue("tipo"), r.FormValue("recorrente") == "on")
		http.Redirect(w, r, "/receitas", http.StatusSeeOther)
	}
}

func HandleDeleteReceita(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		store.DeleteReceita(db, auth.CurrentUser(r).FamilyID, parseInt(r.PathValue("id")))
		http.Redirect(w, r, "/receitas", http.StatusSeeOther)
	}
}

// --- Investimentos ---

func HandleInvestimentos(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		fid := auth.CurrentUser(r).FamilyID
		list, _ := store.GetInvestimentos(db, fid)
		totais, totalGeral, _ := store.GetInvestimentosTotais(db, fid)
		reserva, _ := store.GetReservaEM(db, fid)
		historico, _ := store.GetHistoricoReserva(db, fid)
		render(w, "investimentos", models.InvestimentosPage{
			BasePage:         bp(db, r, "investimentos", "Investimentos"),
			Investimentos:    list,
			Totais:           totais,
			TotalGeral:       totalGeral + reserva, // reserva conta como investimento
			ReservaEM:        reserva,
			HistoricoReserva: historico,
		})
	}
}

func HandleCreateInvestimento(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		r.ParseForm()
		store.CreateInvestimento(db, auth.CurrentUser(r).FamilyID, r.FormValue("instituicao"), r.FormValue("tipo"),
			parseFloat(r.FormValue("valor")), parseDate(r.FormValue("data")), r.FormValue("notas"))
		http.Redirect(w, r, "/investimentos", http.StatusSeeOther)
	}
}

func HandleDeleteInvestimento(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		store.DeleteInvestimento(db, auth.CurrentUser(r).FamilyID, parseInt(r.PathValue("id")))
		http.Redirect(w, r, "/investimentos", http.StatusSeeOther)
	}
}

func HandleAddReservaEM(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		r.ParseForm()
		fid := auth.CurrentUser(r).FamilyID
		tipo := r.FormValue("tipo")
		if tipo == "" {
			tipo = "deposito"
		}
		store.AddReservaEM(db, fid, parseFloat(r.FormValue("valor")), parseDate(r.FormValue("data")), tipo, r.FormValue("notas"))
		http.Redirect(w, r, "/investimentos", http.StatusSeeOther)
	}
}

func HandleUpdateReservaEM(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		r.ParseForm()
		fid := auth.CurrentUser(r).FamilyID
		id := parseInt(r.PathValue("id"))
		tipo := r.FormValue("tipo")
		if tipo == "" {
			tipo = "deposito"
		}
		store.UpdateReservaEM(db, fid, id, parseFloat(r.FormValue("valor")), parseDate(r.FormValue("data")), tipo, r.FormValue("notas"))
		http.Redirect(w, r, "/investimentos", http.StatusSeeOther)
	}
}

func HandleDeleteReservaEM(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		store.DeleteReservaEM(db, auth.CurrentUser(r).FamilyID, parseInt(r.PathValue("id")))
		http.Redirect(w, r, "/investimentos", http.StatusSeeOther)
	}
}

func HandleUpdateInvestimento(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		r.ParseForm()
		fid := auth.CurrentUser(r).FamilyID
		id := parseInt(r.PathValue("id"))
		store.UpdateInvestimento(db, fid, id, r.FormValue("instituicao"), r.FormValue("tipo"),
			parseFloat(r.FormValue("valor")), parseDate(r.FormValue("data")), r.FormValue("notas"))
		http.Redirect(w, r, "/investimentos", http.StatusSeeOther)
	}
}

func HandleUpdateReceita(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		r.ParseForm()
		fid := auth.CurrentUser(r).FamilyID
		id := parseInt(r.PathValue("id"))
		store.UpdateReceita(db, fid, id, r.FormValue("descricao"), parseFloat(r.FormValue("valor")),
			parseDate(r.FormValue("data")), r.FormValue("tipo"), r.FormValue("recorrente") == "on")
		http.Redirect(w, r, "/receitas", http.StatusSeeOther)
	}
}

func HandlePagamentos(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		fid := auth.CurrentUser(r).FamilyID
		mes, mesStr := mesFromRequest(r)
		store.EnsureDespesasMes(db, fid, mes)
		store.AutoFinalizarParcelamentos(db, fid)
		basicas, cartao, vr, _ := store.GetDespesasMes(db, fid, mes, mesStr)
		parcelamentosMes, _ := store.GetParcelamentosVigentesNoMes(db, fid, mes)
		render(w, "pagamentos", models.DespesasPage{
			BasePage:         bp(db, r, "pagamentos", "Pagamentos do Mês"),
			MesStr:           mesStr,
			MesDisplay:       mesDisplay(mesStr),
			MesPrev:          mes.AddDate(0, -1, 0).Format("2006-01"),
			MesNext:          mes.AddDate(0, 1, 0).Format("2006-01"),
			DespesasBasicas:  basicas,
			DespesasCartao:   cartao,
			DespesasVR:       vr,
			ParcelamentosMes: parcelamentosMes,
		})
	}
}

// --- Empréstimos ---

func HandleEmprestimos(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		fid := auth.CurrentUser(r).FamilyID
		list, _ := store.GetEmprestimos(db, fid)
		var totalDevo, totalAReceber float64
		for _, e := range list {
			if e.Pago { continue }
			if e.Direcao == "devo" {
				totalDevo += e.ValorRestante
			} else {
				totalAReceber += e.ValorRestante
			}
		}
		render(w, "emprestimos", models.EmprestimosPage{
			BasePage:      bp(db, r, "emprestimos", "Empréstimos"),
			Emprestimos:   list,
			TotalDevo:     totalDevo,
			TotalAReceber: totalAReceber,
		})
	}
}

func HandleCreateEmprestimo(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		r.ParseForm()
		store.CreateEmprestimo(db, auth.CurrentUser(r).FamilyID, r.FormValue("pessoa"), parseFloat(r.FormValue("valor")),
			r.FormValue("direcao"), parseDate(r.FormValue("data")), r.FormValue("notas"))
		http.Redirect(w, r, "/emprestimos", http.StatusSeeOther)
	}
}

func HandleDeleteEmprestimo(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		store.DeleteEmprestimo(db, auth.CurrentUser(r).FamilyID, parseInt(r.PathValue("id")))
		http.Redirect(w, r, "/emprestimos", http.StatusSeeOther)
	}
}

func HandleToggleEmprestimoPago(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		fid := auth.CurrentUser(r).FamilyID
		id := parseInt(r.PathValue("id"))
		if _, err := store.ToggleEmprestimoPago(db, fid, id); err != nil {
			log.Printf("toggle emprestimo: %v", err)
		}
		http.Redirect(w, r, "/emprestimos", http.StatusSeeOther)
	}
}

func HandlePagamentoEmprestimo(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		r.ParseForm()
		fid := auth.CurrentUser(r).FamilyID
		id := parseInt(r.PathValue("id"))
		valor := parseFloat(r.FormValue("valor"))
		data := parseDate(r.FormValue("data"))
		notas := r.FormValue("notas")
		if valor > 0 {
			if err := store.AddPagamentoEmprestimo(db, fid, id, valor, data, notas); err != nil {
				log.Printf("pagamento emprestimo: %v", err)
			}
		}
		http.Redirect(w, r, "/emprestimos", http.StatusSeeOther)
	}
}

// --- Planejamento ---

func HandlePlanejamento(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		fid := auth.CurrentUser(r).FamilyID
		mes := time.Now()
		mes = time.Date(mes.Year(), mes.Month(), 1, 0, 0, 0, 0, time.UTC)

		store.AutoFinalizarParcelamentos(db, fid)

		var receitaMensal float64
		db.QueryRow(`SELECT COALESCE(SUM(valor),0) FROM receitas WHERE recorrente = true AND family_id = $1`, fid).Scan(&receitaMensal)

		var despesasFixas float64
		db.QueryRow(`SELECT COALESCE(SUM(valor),0) FROM despesas_fixas WHERE ativa = true AND family_id = $1`, fid).Scan(&despesasFixas)

		var despesasParc float64
		db.QueryRow(`
			SELECT COALESCE(SUM(valor_parcela),0) FROM parcelamentos
			WHERE ativo = true AND family_id = $2
			  AND data_inicio <= $1::date
			  AND (EXTRACT(YEAR FROM $1::date) - EXTRACT(YEAR FROM data_inicio)) * 12
			    + (EXTRACT(MONTH FROM $1::date) - EXTRACT(MONTH FROM data_inicio)) + 1 <= total_parcelas`, mes, fid).Scan(&despesasParc)

		despesaMensal := despesasFixas + despesasParc
		sobra := receitaMensal - despesaMensal
		taxaPoup := 0.0
		if receitaMensal > 0 {
			taxaPoup = (sobra / receitaMensal) * 100
		}

		_, totalInvest, _ := store.GetInvestimentosTotais(db, fid)
		reserva, _ := store.GetReservaEM(db, fid)
		patrimonio := totalInvest + reserva
		despesaAnual := despesaMensal * 12
		historico, _ := store.GetHistoricoMeses(db, fid, mes, 6)

		render(w, "planejamento", models.PlanejamentoPage{
			BasePage:        bp(db, r, "planejamento", "Planejamento"),
			ReceitaMensal:   receitaMensal,
			DespesaMensal:   despesaMensal,
			SobraMensal:     sobra,
			TaxaPoupanca:    taxaPoup,
			PatrimonioAtual: patrimonio,
			TotalInvestido:  totalInvest,
			ReservaAtual:    reserva,
			ReservaIdeal6m:  despesaMensal * 6,
			ReservaIdeal12m: despesaMensal * 12,
			DespesaAnual:    despesaAnual,
			FireAgressivo:   despesaAnual * 25,
			FireModerado:    despesaAnual * 28.5,
			FireConservador: despesaAnual * 33.3,
			HistoricoSobra:  historico,
		})
	}
}

// --- Cadastros ---

func HandleCadastros(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		u := auth.CurrentUser(r)
		fid := u.FamilyID
		tab := r.URL.Query().Get("tab")
		if tab == "" { tab = "categorias" }
		categorias, _ := store.GetCategorias(db, fid)
		cartoes, _ := store.GetCartoes(db, fid)

		page := models.CadastrosPage{
			BasePage:   bp(db, r, "cadastros", "Cadastros"),
			TabAtivo:   tab,
			Categorias: categorias,
			Cartoes:    cartoes,
		}

		if tab == "integracao" && u.Admin {
			cfg, err := store.GetIntegracaoConfig(db, "pluggy")
			if err == nil {
				page.PluggyConfig = cfg
				page.PluggyContas, _ = store.GetPluggyItems(db, fid)
				page.PluggyStatus = pluggyHealthCheck(cfg.ServiceURL)
			} else {
				page.PluggyStatus = "nao_configurado"
			}
		}

		render(w, "cadastros", page)
	}
}

func HandleCreateCategoria(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		r.ParseForm()
		fid := auth.CurrentUser(r).FamilyID
		cor := r.FormValue("cor")
		if cor == "" { cor = "#607d8b" }
		store.CreateCategoria(db, fid, r.FormValue("nome"), r.FormValue("grupo"), cor)
		http.Redirect(w, r, "/cadastros?tab=categorias", http.StatusSeeOther)
	}
}

func HandleUpdateCategoria(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		r.ParseForm()
		fid := auth.CurrentUser(r).FamilyID
		cor := r.FormValue("cor")
		if cor == "" { cor = "#607d8b" }
		store.UpdateCategoria(db, fid, parseInt(r.PathValue("id")), r.FormValue("nome"), r.FormValue("grupo"), cor)
		http.Redirect(w, r, "/cadastros?tab=categorias", http.StatusSeeOther)
	}
}

func HandleDeleteCategoria(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		store.DeleteCategoria(db, auth.CurrentUser(r).FamilyID, parseInt(r.PathValue("id")))
		http.Redirect(w, r, "/cadastros?tab=categorias", http.StatusSeeOther)
	}
}

func HandleToggleCategoriaAtiva(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		store.ToggleCategoriaAtiva(db, auth.CurrentUser(r).FamilyID, parseInt(r.PathValue("id")))
		http.Redirect(w, r, "/cadastros?tab=categorias", http.StatusSeeOther)
	}
}

func HandleCreateCartao(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		r.ParseForm()
		fid := auth.CurrentUser(r).FamilyID
		cor := r.FormValue("cor")
		if cor == "" { cor = "#607d8b" }
		store.CreateCartao(db, fid, r.FormValue("nome"), r.FormValue("bandeira"),
			parseFloat(r.FormValue("limite")), cor)
		http.Redirect(w, r, "/cadastros?tab=cartoes", http.StatusSeeOther)
	}
}

func HandleUpdateCartao(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		r.ParseForm()
		fid := auth.CurrentUser(r).FamilyID
		cor := r.FormValue("cor")
		if cor == "" { cor = "#607d8b" }
		store.UpdateCartao(db, fid, parseInt(r.PathValue("id")), r.FormValue("nome"),
			r.FormValue("bandeira"), parseFloat(r.FormValue("limite")), cor)
		http.Redirect(w, r, "/cadastros?tab=cartoes", http.StatusSeeOther)
	}
}

func HandleDeleteCartao(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		store.DeleteCartao(db, auth.CurrentUser(r).FamilyID, parseInt(r.PathValue("id")))
		http.Redirect(w, r, "/cadastros?tab=cartoes", http.StatusSeeOther)
	}
}

func HandleToggleCartaoAtivo(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		store.ToggleCartaoAtivo(db, auth.CurrentUser(r).FamilyID, parseInt(r.PathValue("id")))
		http.Redirect(w, r, "/cadastros?tab=cartoes", http.StatusSeeOther)
	}
}

// --- Usuários ---

func HandleUsuarios(db *sql.DB) http.HandlerFunc {
	return auth.AdminOnly(db, func(w http.ResponseWriter, r *http.Request) {
		users, _ := store.GetUsers(db)
		familias, _ := store.GetFamilies(db)
		render(w, "usuarios", models.UsuariosPage{
			BasePage: bp(db, r, "usuarios", "Usuários"),
			Usuarios: users,
			Familias: familias,
		})
	})
}

func HandleCreateUsuario(db *sql.DB) http.HandlerFunc {
	return auth.AdminOnly(db, func(w http.ResponseWriter, r *http.Request) {
		r.ParseForm()
		nome          := strings.TrimSpace(r.FormValue("nome"))
		email         := strings.TrimSpace(r.FormValue("email"))
		senha         := r.FormValue("senha")
		admin         := r.FormValue("admin") == "on"
		familyOpt     := r.FormValue("family_opt")
		familyIDStr   := r.FormValue("family_id")
		newFamilyName := strings.TrimSpace(r.FormValue("new_family_name"))

		if nome == "" || email == "" || senha == "" {
			http.Redirect(w, r, "/usuarios?erro=campos", http.StatusSeeOther)
			return
		}
		hash, err := store.HashSenha(senha)
		if err != nil {
			http.Redirect(w, r, "/usuarios?erro=hash", http.StatusSeeOther)
			return
		}

		var familyID int
		if familyOpt == "nova" {
			if newFamilyName == "" { newFamilyName = nome }
			familyID, err = store.CreateFamily(db, newFamilyName)
			if err != nil {
				log.Printf("create family: %v", err)
				http.Redirect(w, r, "/usuarios?erro=familia", http.StatusSeeOther)
				return
			}
		} else {
			familyID = parseInt(familyIDStr)
			if familyID == 0 {
				db.QueryRow(`SELECT id FROM families WHERE nome='Família Padrão'`).Scan(&familyID)
			}
		}

		if err := store.CreateUser(db, nome, email, hash, admin, familyID); err != nil {
			log.Printf("create user: %v", err)
			http.Redirect(w, r, "/usuarios?erro=email", http.StatusSeeOther)
			return
		}
		http.Redirect(w, r, "/usuarios", http.StatusSeeOther)
	})
}

func HandleChangeUserFamily(db *sql.DB) http.HandlerFunc {
	return auth.AdminOnly(db, func(w http.ResponseWriter, r *http.Request) {
		r.ParseForm()
		uid := parseInt(r.PathValue("id"))
		fid := parseInt(r.FormValue("family_id"))
		if fid > 0 { store.SetUserFamily(db, uid, fid) }
		http.Redirect(w, r, "/usuarios", http.StatusSeeOther)
	})
}

func HandleToggleUsuarioAdmin(db *sql.DB) http.HandlerFunc {
	return auth.AdminOnly(db, func(w http.ResponseWriter, r *http.Request) {
		id := parseInt(r.PathValue("id"))
		if id == auth.CurrentUser(r).ID {
			http.Redirect(w, r, "/usuarios", http.StatusSeeOther)
			return
		}
		store.ToggleUserAdmin(db, id)
		http.Redirect(w, r, "/usuarios", http.StatusSeeOther)
	})
}

func HandleToggleUsuarioAtivo(db *sql.DB) http.HandlerFunc {
	return auth.AdminOnly(db, func(w http.ResponseWriter, r *http.Request) {
		id := parseInt(r.PathValue("id"))
		if id == auth.CurrentUser(r).ID {
			http.Redirect(w, r, "/usuarios", http.StatusSeeOther)
			return
		}
		store.ToggleUserAtivo(db, id)
		http.Redirect(w, r, "/usuarios", http.StatusSeeOther)
	})
}

func HandleResetSenha(db *sql.DB) http.HandlerFunc {
	return auth.AdminOnly(db, func(w http.ResponseWriter, r *http.Request) {
		r.ParseForm()
		id    := parseInt(r.PathValue("id"))
		senha := r.FormValue("nova_senha")
		if senha == "" {
			http.Redirect(w, r, "/usuarios", http.StatusSeeOther)
			return
		}
		if hash, err := store.HashSenha(senha); err == nil {
			store.ResetSenha(db, id, hash)
		}
		http.Redirect(w, r, "/usuarios", http.StatusSeeOther)
	})
}

func HandleDeleteUsuario(db *sql.DB) http.HandlerFunc {
	return auth.AdminOnly(db, func(w http.ResponseWriter, r *http.Request) {
		id := parseInt(r.PathValue("id"))
		if id == auth.CurrentUser(r).ID {
			http.Redirect(w, r, "/usuarios", http.StatusSeeOther)
			return
		}
		store.DeleteUser(db, id)
		http.Redirect(w, r, "/usuarios", http.StatusSeeOther)
	})
}

func HandleToggleFamilyAdminFlag(db *sql.DB) http.HandlerFunc {
	return auth.AdminOnly(db, func(w http.ResponseWriter, r *http.Request) {
		id := parseInt(r.PathValue("id"))
		if id != auth.CurrentUser(r).ID { store.ToggleFamilyAdmin(db, id) }
		http.Redirect(w, r, "/usuarios", http.StatusSeeOther)
	})
}

// --- Famílias ---

func HandleFamilias(db *sql.DB) http.HandlerFunc {
	return auth.AdminOnly(db, func(w http.ResponseWriter, r *http.Request) {
		familias, _ := store.GetFamilies(db)
		render(w, "familias", models.FamiliasPage{
			BasePage: bp(db, r, "familias", "Famílias"),
			Familias: familias,
		})
	})
}

func HandleCreateFamilia(db *sql.DB) http.HandlerFunc {
	return auth.AdminOnly(db, func(w http.ResponseWriter, r *http.Request) {
		r.ParseForm()
		nome := strings.TrimSpace(r.FormValue("nome"))
		if nome != "" {
			if _, err := store.CreateFamily(db, nome); err != nil {
				log.Printf("create family: %v", err)
			}
		}
		http.Redirect(w, r, "/familias", http.StatusSeeOther)
	})
}

func HandleRenameFamilia(db *sql.DB) http.HandlerFunc {
	return auth.AdminOnly(db, func(w http.ResponseWriter, r *http.Request) {
		r.ParseForm()
		id := parseInt(r.PathValue("id"))
		nome := strings.TrimSpace(r.FormValue("nome"))
		if nome != "" { store.RenameFamily(db, id, nome) }
		http.Redirect(w, r, "/familias", http.StatusSeeOther)
	})
}

func HandleDeleteFamilia(db *sql.DB) http.HandlerFunc {
	return auth.AdminOnly(db, func(w http.ResponseWriter, r *http.Request) {
		id := parseInt(r.PathValue("id"))
		if err := store.DeleteFamily(db, id); err != nil {
			log.Printf("delete family: %v", err)
		}
		http.Redirect(w, r, "/familias", http.StatusSeeOther)
	})
}

// --- Minha Família ---

func HandleMeuTime(db *sql.DB) http.HandlerFunc {
	return auth.FamilyAdminOnly(db, func(w http.ResponseWriter, r *http.Request) {
		cu := auth.CurrentUser(r)
		membros, _ := store.GetFamilyMembers(db, cu.FamilyID)
		for i := range membros {
			membros[i].BlockedScreens, _ = store.GetBlockedScreens(db, membros[i].ID)
		}
		var erro, sucesso string
		if e := r.URL.Query().Get("erro"); e != "" {
			switch e {
			case "campos":
				erro = "Preencha nome, e-mail e senha."
			case "email":
				erro = "E-mail já cadastrado ou inválido."
			case "hash":
				erro = "Erro ao processar senha."
			default:
				erro = e
			}
		}
		if r.URL.Query().Get("ok") == "1" {
			sucesso = "Operação realizada com sucesso."
		}
		render(w, "minha-familia", models.MeuTimePage{
			BasePage: bp(db, r, "minha-familia", "Minha Família"),
			Membros:  membros,
			Telas:    TelasDispo,
			Erro:     erro,
			Sucesso:  sucesso,
		})
	})
}

func HandleToggleScreenPermission(db *sql.DB) http.HandlerFunc {
	return auth.FamilyAdminOnly(db, func(w http.ResponseWriter, r *http.Request) {
		cu := auth.CurrentUser(r)
		idStr := r.PathValue("id")
		tela := r.PathValue("tela")
		targetID, err := strconv.Atoi(idStr)
		if err != nil {
			http.Redirect(w, r, "/minha-familia?ok=1", http.StatusSeeOther)
			return
		}
		membros, _ := store.GetFamilyMembers(db, cu.FamilyID)
		var targetOK bool
		for _, m := range membros {
			if m.ID == targetID && !m.Admin {
				targetOK = true
				break
			}
		}
		if !targetOK {
			http.Redirect(w, r, "/minha-familia?ok=1", http.StatusSeeOther)
			return
		}
		for _, t := range TelasDispo {
			if t.Key == tela {
				store.ToggleScreenPermission(db, targetID, tela)
				break
			}
		}
		http.Redirect(w, r, "/minha-familia?ok=1", http.StatusSeeOther)
	})
}

func HandleMeuTimeCreateMembro(db *sql.DB) http.HandlerFunc {
	return auth.FamilyAdminOnly(db, func(w http.ResponseWriter, r *http.Request) {
		r.ParseForm()
		cu := auth.CurrentUser(r)
		nome  := strings.TrimSpace(r.FormValue("nome"))
		email := strings.TrimSpace(r.FormValue("email"))
		senha := r.FormValue("senha")
		if nome == "" || email == "" || senha == "" {
			http.Redirect(w, r, "/minha-familia?erro=campos", http.StatusSeeOther)
			return
		}
		hash, err := store.HashSenha(senha)
		if err != nil {
			http.Redirect(w, r, "/minha-familia?erro=hash", http.StatusSeeOther)
			return
		}
		if err := store.CreateUser(db, nome, email, hash, false, cu.FamilyID); err != nil {
			log.Printf("minha-familia create user: %v", err)
			http.Redirect(w, r, "/minha-familia?erro=email", http.StatusSeeOther)
			return
		}
		http.Redirect(w, r, "/minha-familia?ok=1", http.StatusSeeOther)
	})
}

func HandleMeuTimeToggleAtivo(db *sql.DB) http.HandlerFunc {
	return auth.FamilyAdminOnly(db, func(w http.ResponseWriter, r *http.Request) {
		cu := auth.CurrentUser(r)
		id := parseInt(r.PathValue("id"))
		if id == cu.ID {
			http.Redirect(w, r, "/minha-familia", http.StatusSeeOther)
			return
		}
		membros, _ := store.GetFamilyMembers(db, cu.FamilyID)
		for _, m := range membros {
			if m.ID == id {
				store.ToggleUserAtivo(db, id)
				break
			}
		}
		http.Redirect(w, r, "/minha-familia?ok=1", http.StatusSeeOther)
	})
}

func HandleMeuTimeResetSenha(db *sql.DB) http.HandlerFunc {
	return auth.FamilyAdminOnly(db, func(w http.ResponseWriter, r *http.Request) {
		r.ParseForm()
		cu := auth.CurrentUser(r)
		id    := parseInt(r.PathValue("id"))
		senha := r.FormValue("nova_senha")
		if senha == "" {
			http.Redirect(w, r, "/minha-familia", http.StatusSeeOther)
			return
		}
		membros, _ := store.GetFamilyMembers(db, cu.FamilyID)
		for _, m := range membros {
			if m.ID == id {
				if m.Admin && !cu.Admin {
					http.Redirect(w, r, "/minha-familia", http.StatusSeeOther)
					return
				}
				if hash, err := store.HashSenha(senha); err == nil {
					store.ResetSenha(db, id, hash)
				}
				break
			}
		}
		http.Redirect(w, r, "/minha-familia?ok=1", http.StatusSeeOther)
	})
}

func HandleMinhaSenha(db *sql.DB) http.HandlerFunc {
	return auth.Protected(db, func(w http.ResponseWriter, r *http.Request) {
		r.ParseForm()
		cu := auth.CurrentUser(r)
		senha := r.FormValue("nova_senha")
		if senha == "" {
			http.Redirect(w, r, "/minha-familia", http.StatusSeeOther)
			return
		}
		if hash, err := store.HashSenha(senha); err == nil {
			store.ResetSenha(db, cu.ID, hash)
		}
		http.Redirect(w, r, "/minha-familia?ok=1", http.StatusSeeOther)
	})
}

// ---------------------------------------------------------------------------
// Pluggy Integration Admin Handlers
// ---------------------------------------------------------------------------

func pluggyHealthCheck(serviceURL string) string {
	if serviceURL == "" {
		return "nao_configurado"
	}
	client := &http.Client{Timeout: 3 * time.Second}
	resp, err := client.Get(serviceURL + "/api/pluggy/status")
	if err != nil || resp.StatusCode != http.StatusOK {
		return "offline"
	}
	defer resp.Body.Close()
	return "online"
}

func pluggyProxy(db *sql.DB, method, path string, body any) ([]byte, int, error) {
	cfg, err := store.GetIntegracaoConfig(db, "pluggy")
	if err != nil || !cfg.Ativo {
		return nil, http.StatusServiceUnavailable, fmt.Errorf("pluggy nao configurado")
	}

	var reqBody io.Reader
	if body != nil {
		data, _ := json.Marshal(body)
		reqBody = bytes.NewReader(data)
	}

	req, err := http.NewRequest(method, cfg.ServiceURL+path, reqBody)
	if err != nil {
		return nil, http.StatusInternalServerError, err
	}
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, http.StatusBadGateway, fmt.Errorf("pluggy-service indisponivel: %w", err)
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)
	return respBody, resp.StatusCode, nil
}

func HandleSavePluggyConfig(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		r.ParseForm()
		cfg := models.IntegracaoConfig{
			Integracao:   "pluggy",
			ClientID:     strings.TrimSpace(r.FormValue("client_id")),
			ClientSecret: strings.TrimSpace(r.FormValue("client_secret")),
			ServiceURL:   strings.TrimSpace(r.FormValue("service_url")),
			Ativo:        r.FormValue("ativo") == "on",
		}
		if err := store.SaveIntegracaoConfig(db, cfg); err != nil {
			log.Printf("Erro ao salvar config pluggy: %v", err)
		}
		http.Redirect(w, r, "/cadastros?tab=integracao", http.StatusSeeOther)
	}
}

func HandlePluggyConnectToken(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		body, status, err := pluggyProxy(db, "POST", "/api/pluggy/connect-token", nil)
		if err != nil {
			http.Error(w, err.Error(), status)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(status)
		w.Write(body)
	}
}

func HandlePluggyRegisterItem(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		u := auth.CurrentUser(r)
		var payload struct {
			ItemID string `json:"item_id"`
		}
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil || payload.ItemID == "" {
			http.Error(w, "item_id obrigatorio", http.StatusBadRequest)
			return
		}

		// Register item in pluggy-service
		body, status, err := pluggyProxy(db, "POST", "/api/pluggy/items", map[string]any{
			"item_id":   payload.ItemID,
			"family_id": u.FamilyID,
		})
		if err != nil {
			http.Error(w, err.Error(), status)
			return
		}

		// Also save locally
		store.CreatePluggyItem(db, u.FamilyID, payload.ItemID, "")

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(status)
		w.Write(body)
	}
}

func HandlePluggyDisconnect(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		u := auth.CurrentUser(r)
		itemID := r.PathValue("item_id")
		store.DeletePluggyItem(db, u.FamilyID, itemID)
		http.Redirect(w, r, "/cadastros?tab=integracao", http.StatusSeeOther)
	}
}

func HandlePluggySync(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		itemID := r.PathValue("item_id")
		_, status, err := pluggyProxy(db, "POST", "/api/pluggy/sync/"+itemID, nil)
		if err != nil {
			http.Error(w, err.Error(), status)
			return
		}
		http.Redirect(w, r, "/cadastros?tab=integracao", http.StatusSeeOther)
	}
}

// HandlePluggyWebhook faz proxy do webhook externo para o pluggy-service interno.
// Rota publica (sem auth) — a validacao e feita por token no pluggy-service.
func HandlePluggyWebhook(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		cfg, err := store.GetIntegracaoConfig(db, "pluggy")
		if err != nil || cfg.ServiceURL == "" {
			cfg = &models.IntegracaoConfig{ServiceURL: "http://pluggy-service:8081"}
		}

		// Forward token from query string
		targetURL := cfg.ServiceURL + "/api/pluggy/webhook"
		if token := r.URL.Query().Get("token"); token != "" {
			targetURL += "?token=" + token
		}

		proxyReq, err := http.NewRequest("POST", targetURL, r.Body)
		if err != nil {
			http.Error(w, "proxy error", http.StatusBadGateway)
			return
		}
		proxyReq.Header.Set("Content-Type", "application/json")
		// Forward webhook token header if present
		if h := r.Header.Get("X-Webhook-Token"); h != "" {
			proxyReq.Header.Set("X-Webhook-Token", h)
		}

		client := &http.Client{Timeout: 30 * time.Second}
		resp, err := client.Do(proxyReq)
		if err != nil {
			log.Printf("[webhook-proxy] erro: %v", err)
			http.Error(w, "pluggy-service indisponivel", http.StatusBadGateway)
			return
		}
		defer resp.Body.Close()

		w.WriteHeader(resp.StatusCode)
		io.Copy(w, resp.Body)
	}
}
