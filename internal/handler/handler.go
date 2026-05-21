package handler

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"fincontrol/internal/auth"
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
	pages := []string{"dashboard", "despesas", "receitas", "investimentos", "planejamento", "emprestimos", "usuarios", "familias", "cadastros", "minha-familia"}
	for _, page := range pages {
		t := template.New("").Funcs(buildFuncMap())
		template.Must(t.ParseFiles("templates/base.html", "templates/"+page+".html"))
		tmplPages[page] = t
	}
	lt := template.Must(template.New("login").Funcs(buildFuncMap()).ParseFiles("templates/login.html"))
	tmplPages["login"] = lt
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
	s = strings.ReplaceAll(s, ".", "")
	s = strings.ReplaceAll(s, ",", ".")
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
		reservaEM, _ := store.GetReservaEM(db, fid)
		historico, _ := store.GetHistoricoMeses(db, fid, mes, 6)

		var totalDespesas float64
		for _, d := range basicas { totalDespesas += d.Valor }
		for _, d := range cartao  { totalDespesas += d.Valor }
		for _, d := range vr      { totalDespesas += d.Valor }
		for _, p := range parcelamentos { totalDespesas += p.ValorParcela }

		render(w, "dashboard", models.DashboardData{
			BasePage:        bp(db, r, "dashboard", "Dashboard"),
			Mes:             mesStr,
			MesDisplay:      mesDisplay(mesStr),
			MesPrev:         prevMes,
			MesNext:         nextMes,
			TotalReceitas:   totalReceitas,
			TotalDespesas:   totalDespesas,
			Sobra:           totalReceitas - totalDespesas,
			TotalInvestido:  totalInvestido,
			ReservaEM:       reservaEM,
			DespesasBasicas: basicas,
			DespesasCartao:  cartao,
			DespesasVR:      vr,
			Parcelamentos:   parcelamentos,
			Historico:       historico,
		})
	}
}

// --- Toggle pago (HTMX) ---

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
		renderPartial(w, "dashboard", "toggle-btn", d)
	}
}

// --- Despesas ---

func HandleDespesas(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		fid := auth.CurrentUser(r).FamilyID
		tab := r.URL.Query().Get("tab")
		if tab == "" { tab = "fixas" }
		despesas, _ := store.GetDespesasFixas(db, fid)
		parcelamentos, _ := store.GetParcelamentos(db, fid, false)
		categorias, _ := store.GetCategorias(db, fid)
		cartoes, _ := store.GetCartoes(db, fid)

		render(w, "despesas", models.DespesasPage{
			BasePage:      bp(db, r, "despesas", "Despesas"),
			TabAtivo:      tab,
			Despesas:      despesas,
			Parcelamentos: parcelamentos,
			Categorias:    categorias,
			Cartoes:       cartoes,
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
		store.CreateParcelamento(db, auth.CurrentUser(r).FamilyID,
			r.FormValue("descricao"), r.FormValue("cartao"),
			parseFloat(r.FormValue("valor_parcela")),
			parseInt(r.FormValue("parcela_atual")),
			parseInt(r.FormValue("total_parcelas")),
			parseDate(r.FormValue("data_inicio")))
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
			TotalGeral:       totalGeral,
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
		store.AddReservaEM(db, auth.CurrentUser(r).FamilyID, parseFloat(r.FormValue("valor")), r.FormValue("notas"))
		http.Redirect(w, r, "/investimentos", http.StatusSeeOther)
	}
}

func HandleDeleteReservaEM(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		store.DeleteReservaEM(db, auth.CurrentUser(r).FamilyID, parseInt(r.PathValue("id")))
		http.Redirect(w, r, "/investimentos", http.StatusSeeOther)
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
		fid := auth.CurrentUser(r).FamilyID
		tab := r.URL.Query().Get("tab")
		if tab == "" { tab = "categorias" }
		categorias, _ := store.GetCategorias(db, fid)
		cartoes, _ := store.GetCartoes(db, fid)
		render(w, "cadastros", models.CadastrosPage{
			BasePage:   bp(db, r, "cadastros", "Cadastros"),
			TabAtivo:   tab,
			Categorias: categorias,
			Cartoes:    cartoes,
		})
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
