package main

import (
	"encoding/json"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"golang.org/x/crypto/bcrypt"
)

// --- Template rendering ---

var tmplPages = map[string]*template.Template{}

func buildFuncMap() template.FuncMap {
	return template.FuncMap{
		"brl":        formatBRL,
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
	}
}

func initTemplates() {
	pages := []string{"dashboard", "despesas", "receitas", "investimentos", "planejamento", "usuarios"}
	for _, page := range pages {
		t := template.New("").Funcs(buildFuncMap())
		template.Must(t.ParseFiles("templates/base.html", "templates/"+page+".html"))
		tmplPages[page] = t
	}
	// Login é standalone — não usa base.html
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

func bp(r *http.Request, active, title string) BasePage {
	return BasePage{CurrentUser: currentUser(r), Active: active, Title: title}
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
	if neg {
		v = -v
	}
	cents := int64(v*100+0.5) % 100
	whole := int64(v)
	s := strconv.FormatInt(whole, 10)
	if len(s) > 3 {
		var b strings.Builder
		for i, c := range s {
			if i > 0 && (len(s)-i)%3 == 0 {
				b.WriteRune('.')
			}
			b.WriteRune(c)
		}
		s = b.String()
	}
	result := fmt.Sprintf("R$ %s,%02d", s, cents)
	if neg {
		result = "-" + result
	}
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
	if err != nil {
		return time.Now()
	}
	return t
}

// --- Login / Logout ---

func handleLogin(app *App) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Se já está autenticado, vai para o dashboard
		if u := authUser(app, r); u != nil && u.Ativo {
			http.Redirect(w, r, "/", http.StatusSeeOther)
			return
		}
		if r.Method == http.MethodGet {
			erro := ""
			if r.URL.Query().Get("erro") != "" {
				erro = "Email ou senha incorretos."
			}
			renderLogin(w, LoginPage{Title: "Login — FinBertoldi", Erro: erro})
			return
		}
		// POST
		r.ParseForm()
		email := strings.TrimSpace(r.FormValue("email"))
		senha := r.FormValue("senha")

		user, hash, err := dbGetUserByEmail(app.db, email)
		if err != nil || !user.Ativo {
			http.Redirect(w, r, "/login?erro=1", http.StatusSeeOther)
			return
		}
		if bcrypt.CompareHashAndPassword([]byte(hash), []byte(senha)) != nil {
			http.Redirect(w, r, "/login?erro=1", http.StatusSeeOther)
			return
		}

		token := generateToken()
		expiry := time.Now().Add(7 * 24 * time.Hour)
		if err := dbCreateSession(app.db, user.ID, token, expiry); err != nil {
			log.Printf("create session: %v", err)
		}
		setSessionCookie(w, token, expiry)
		http.Redirect(w, r, "/", http.StatusSeeOther)
	}
}

func handleLogout(app *App) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if c, err := r.Cookie("session"); err == nil {
			dbDeleteSession(app.db, c.Value)
		}
		clearSessionCookie(w)
		http.Redirect(w, r, "/login", http.StatusSeeOther)
	}
}

// --- Dashboard ---

func handleDashboard(app *App) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		mes, mesStr := mesFromRequest(r)
		prevMes := mes.AddDate(0, -1, 0).Format("2006-01")
		nextMes := mes.AddDate(0, 1, 0).Format("2006-01")

		if err := dbEnsureDespesasMes(app.db, mes); err != nil {
			log.Printf("ensure despesas mes: %v", err)
		}
		// Auto-finaliza parcelamentos que já passaram do total
		if err := dbAutoFinalizarParcelamentos(app.db); err != nil {
			log.Printf("auto finalizar parcelamentos: %v", err)
		}

		basicas, cartao, vr, err := dbGetDespesasMes(app.db, mes, mesStr)
		if err != nil {
			log.Printf("get despesas mes: %v", err)
		}
		// Parcelamentos vigentes nesse mês específico
		parcelamentos, _ := dbGetParcelamentosVigentesNoMes(app.db, mes)
		totalReceitas, _ := dbGetReceitasMes(app.db, mes)
		_, totalInvestido, _ := dbGetInvestimentosTotais(app.db)
		reservaEM, _ := dbGetReservaEM(app.db)
		historico, _ := dbGetHistoricoMeses(app.db, mes, 6)

		var totalDespesas float64
		for _, d := range basicas { totalDespesas += d.Valor }
		for _, d := range cartao  { totalDespesas += d.Valor }
		for _, d := range vr      { totalDespesas += d.Valor }
		for _, p := range parcelamentos { totalDespesas += p.ValorParcela }

		render(w, "dashboard", DashboardData{
			BasePage:        bp(r, "dashboard", "Dashboard"),
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

func handleToggleDespesaPago(app *App) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := parseInt(r.PathValue("id"))
		mes, mesStr := mesFromRequest(r)

		pago, err := dbToggleDespesaMesPago(app.db, id, mes)
		if err != nil {
			http.Error(w, err.Error(), 500)
			return
		}
		var d DespesaMes
		app.db.QueryRow(`SELECT id, nome, valor, categoria FROM despesas_fixas WHERE id=$1`, id).
			Scan(&d.ID, &d.Nome, &d.Valor, &d.Categoria)
		d.Mes = mesStr
		d.Pago = pago
		renderPartial(w, "dashboard", "toggle-btn", d)
	}
}

// --- Despesas ---

func handleDespesas(app *App) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		tab := r.URL.Query().Get("tab")
		if tab == "" {
			tab = "fixas"
		}
		despesas, _ := dbGetDespesasFixas(app.db)
		parcelamentos, _ := dbGetParcelamentos(app.db, false)
		emprestimos, _ := dbGetEmprestimos(app.db)

		var totalDevo, totalAReceber float64
		for _, e := range emprestimos {
			if e.Pago { continue }
			if e.Direcao == "devo" {
				totalDevo += e.Valor
			} else {
				totalAReceber += e.Valor
			}
		}
		render(w, "despesas", DespesasPage{
			BasePage:      bp(r, "despesas", "Despesas"),
			TabAtivo:      tab,
			Despesas:      despesas,
			Parcelamentos: parcelamentos,
			Emprestimos:   emprestimos,
			TotalDevo:     totalDevo,
			TotalAReceber: totalAReceber,
		})
	}
}

func handleCreateDespesa(app *App) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		r.ParseForm()
		dbCreateDespesa(app.db, r.FormValue("nome"), parseFloat(r.FormValue("valor")), r.FormValue("categoria"))
		http.Redirect(w, r, "/despesas", http.StatusSeeOther)
	}
}

func handleUpdateDespesa(app *App) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		r.ParseForm()
		dbUpdateDespesa(app.db, parseInt(r.PathValue("id")),
			r.FormValue("nome"), parseFloat(r.FormValue("valor")), r.FormValue("categoria"))
		http.Redirect(w, r, "/despesas", http.StatusSeeOther)
	}
}

func handleDeleteDespesa(app *App) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		dbDeleteDespesa(app.db, parseInt(r.PathValue("id")))
		http.Redirect(w, r, "/despesas", http.StatusSeeOther)
	}
}

func handleToggleDespesaAtiva(app *App) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		dbToggleDespesaAtiva(app.db, parseInt(r.PathValue("id")))
		http.Redirect(w, r, "/despesas", http.StatusSeeOther)
	}
}

// --- Parcelamentos ---

func handleParcelamentos(app *App) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "/despesas?tab=parcelamentos", http.StatusSeeOther)
	}
}

func handleCreateParcelamento(app *App) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		r.ParseForm()
		dbCreateParcelamento(app.db,
			r.FormValue("descricao"), r.FormValue("cartao"),
			parseFloat(r.FormValue("valor_parcela")),
			parseInt(r.FormValue("parcela_atual")),
			parseInt(r.FormValue("total_parcelas")),
			parseDate(r.FormValue("data_inicio")))
		http.Redirect(w, r, "/despesas?tab=parcelamentos", http.StatusSeeOther)
	}
}

func handleUpdateParcelamento(app *App) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		r.ParseForm()
		dbUpdateParcelamento(app.db, parseInt(r.PathValue("id")),
			parseInt(r.FormValue("parcela_atual")), r.FormValue("ativo") == "true")
		http.Redirect(w, r, "/despesas?tab=parcelamentos", http.StatusSeeOther)
	}
}

func handleDeleteParcelamento(app *App) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		dbDeleteParcelamento(app.db, parseInt(r.PathValue("id")))
		http.Redirect(w, r, "/despesas?tab=parcelamentos", http.StatusSeeOther)
	}
}

// --- Receitas ---

func handleReceitas(app *App) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		list, _ := dbGetReceitas(app.db)
		render(w, "receitas", ReceitasPage{BasePage: bp(r, "receitas", "Receitas"), Receitas: list})
	}
}

func handleCreateReceita(app *App) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		r.ParseForm()
		dbCreateReceita(app.db, r.FormValue("descricao"), parseFloat(r.FormValue("valor")),
			parseDate(r.FormValue("data")), r.FormValue("tipo"), r.FormValue("recorrente") == "on")
		http.Redirect(w, r, "/receitas", http.StatusSeeOther)
	}
}

func handleDeleteReceita(app *App) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		dbDeleteReceita(app.db, parseInt(r.PathValue("id")))
		http.Redirect(w, r, "/receitas", http.StatusSeeOther)
	}
}

// --- Investimentos ---

func handleInvestimentos(app *App) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		list, _ := dbGetInvestimentos(app.db)
		totais, totalGeral, _ := dbGetInvestimentosTotais(app.db)
		reserva, _ := dbGetReservaEM(app.db)
		historico, _ := dbGetHistoricoReserva(app.db)
		render(w, "investimentos", InvestimentosPage{
			BasePage:         bp(r, "investimentos", "Investimentos"),
			Investimentos:    list,
			Totais:           totais,
			TotalGeral:       totalGeral,
			ReservaEM:        reserva,
			HistoricoReserva: historico,
		})
	}
}

func handleCreateInvestimento(app *App) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		r.ParseForm()
		dbCreateInvestimento(app.db, r.FormValue("instituicao"), r.FormValue("tipo"),
			parseFloat(r.FormValue("valor")), parseDate(r.FormValue("data")), r.FormValue("notas"))
		http.Redirect(w, r, "/investimentos", http.StatusSeeOther)
	}
}

func handleDeleteInvestimento(app *App) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		dbDeleteInvestimento(app.db, parseInt(r.PathValue("id")))
		http.Redirect(w, r, "/investimentos", http.StatusSeeOther)
	}
}

func handleAddReservaEM(app *App) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		r.ParseForm()
		dbAddReservaEM(app.db, parseFloat(r.FormValue("valor")), r.FormValue("notas"))
		http.Redirect(w, r, "/investimentos", http.StatusSeeOther)
	}
}

func handleDeleteReservaEM(app *App) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		dbDeleteReservaEM(app.db, parseInt(r.PathValue("id")))
		http.Redirect(w, r, "/investimentos", http.StatusSeeOther)
	}
}

// --- Empréstimos ---

func handleEmprestimos(app *App) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "/despesas?tab=emprestimos", http.StatusSeeOther)
	}
}

func handleCreateEmprestimo(app *App) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		r.ParseForm()
		dbCreateEmprestimo(app.db, r.FormValue("pessoa"), parseFloat(r.FormValue("valor")),
			r.FormValue("direcao"), parseDate(r.FormValue("data")), r.FormValue("notas"))
		http.Redirect(w, r, "/despesas?tab=emprestimos", http.StatusSeeOther)
	}
}

func handleDeleteEmprestimo(app *App) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		dbDeleteEmprestimo(app.db, parseInt(r.PathValue("id")))
		http.Redirect(w, r, "/despesas?tab=emprestimos", http.StatusSeeOther)
	}
}

func handleToggleEmprestimoPago(app *App) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := parseInt(r.PathValue("id"))
		pago, err := dbToggleEmprestimoPago(app.db, id)
		if err != nil {
			http.Error(w, err.Error(), 500)
			return
		}
		var e Emprestimo
		app.db.QueryRow(`SELECT id, pessoa, valor, direcao, pago FROM emprestimos WHERE id=$1`, id).
			Scan(&e.ID, &e.Pessoa, &e.Valor, &e.Direcao, &e.Pago)
		e.Pago = pago
		renderPartial(w, "despesas", "emprestimo-toggle-btn", e)
	}
}

// --- Planejamento ---

func handlePlanejamento(app *App) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		mes := time.Now()
		mes = time.Date(mes.Year(), mes.Month(), 1, 0, 0, 0, 0, time.UTC)

		// Auto-finaliza parcelamentos vencidos
		dbAutoFinalizarParcelamentos(app.db)

		// Receitas recorrentes apenas (representam o ganho mensal estável)
		var receitaMensal float64
		app.db.QueryRow(`SELECT COALESCE(SUM(valor),0) FROM receitas WHERE recorrente = true`).Scan(&receitaMensal)

		// Despesas fixas ativas
		var despesasFixas float64
		app.db.QueryRow(`SELECT COALESCE(SUM(valor),0) FROM despesas_fixas WHERE ativa = true`).Scan(&despesasFixas)

		// Parcelamentos vigentes no mês corrente
		var despesasParc float64
		app.db.QueryRow(`
			SELECT COALESCE(SUM(valor_parcela),0) FROM parcelamentos
			WHERE ativo = true
			  AND data_inicio <= $1
			  AND (EXTRACT(YEAR FROM $1) - EXTRACT(YEAR FROM data_inicio)) * 12
			    + (EXTRACT(MONTH FROM $1) - EXTRACT(MONTH FROM data_inicio)) + 1 <= total_parcelas`, mes).Scan(&despesasParc)

		despesaMensal := despesasFixas + despesasParc
		sobra := receitaMensal - despesaMensal
		taxaPoup := 0.0
		if receitaMensal > 0 {
			taxaPoup = (sobra / receitaMensal) * 100
		}

		_, totalInvest, _ := dbGetInvestimentosTotais(app.db)
		reserva, _ := dbGetReservaEM(app.db)
		patrimonio := totalInvest + reserva

		despesaAnual := despesaMensal * 12

		historico, _ := dbGetHistoricoMeses(app.db, mes, 6)

		render(w, "planejamento", PlanejamentoPage{
			BasePage:        bp(r, "planejamento", "Planejamento"),
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

// --- Usuários ---

func handleUsuarios(app *App) http.HandlerFunc {
	return adminOnly(app, func(w http.ResponseWriter, r *http.Request) {
		users, _ := dbGetUsers(app.db)
		render(w, "usuarios", UsuariosPage{
			BasePage: bp(r, "usuarios", "Usuários"),
			Usuarios: users,
		})
	})
}

func handleCreateUsuario(app *App) http.HandlerFunc {
	return adminOnly(app, func(w http.ResponseWriter, r *http.Request) {
		r.ParseForm()
		nome  := strings.TrimSpace(r.FormValue("nome"))
		email := strings.TrimSpace(r.FormValue("email"))
		senha := r.FormValue("senha")
		admin := r.FormValue("admin") == "on"

		if nome == "" || email == "" || senha == "" {
			http.Redirect(w, r, "/usuarios?erro=campos", http.StatusSeeOther)
			return
		}
		hash, err := hashSenha(senha)
		if err != nil {
			http.Redirect(w, r, "/usuarios?erro=hash", http.StatusSeeOther)
			return
		}
		if err := dbCreateUser(app.db, nome, email, hash, admin); err != nil {
			log.Printf("create user: %v", err)
			http.Redirect(w, r, "/usuarios?erro=email", http.StatusSeeOther)
			return
		}
		http.Redirect(w, r, "/usuarios", http.StatusSeeOther)
	})
}

func handleToggleUsuarioAdmin(app *App) http.HandlerFunc {
	return adminOnly(app, func(w http.ResponseWriter, r *http.Request) {
		id := parseInt(r.PathValue("id"))
		if id == currentUser(r).ID {
			http.Redirect(w, r, "/usuarios", http.StatusSeeOther)
			return
		}
		dbToggleUserAdmin(app.db, id)
		http.Redirect(w, r, "/usuarios", http.StatusSeeOther)
	})
}

func handleToggleUsuarioAtivo(app *App) http.HandlerFunc {
	return adminOnly(app, func(w http.ResponseWriter, r *http.Request) {
		id := parseInt(r.PathValue("id"))
		if id == currentUser(r).ID {
			http.Redirect(w, r, "/usuarios", http.StatusSeeOther)
			return
		}
		dbToggleUserAtivo(app.db, id)
		http.Redirect(w, r, "/usuarios", http.StatusSeeOther)
	})
}

func handleResetSenha(app *App) http.HandlerFunc {
	return adminOnly(app, func(w http.ResponseWriter, r *http.Request) {
		r.ParseForm()
		id    := parseInt(r.PathValue("id"))
		senha := r.FormValue("nova_senha")
		if senha == "" {
			http.Redirect(w, r, "/usuarios", http.StatusSeeOther)
			return
		}
		hash, err := hashSenha(senha)
		if err == nil {
			dbResetSenha(app.db, id, hash)
		}
		http.Redirect(w, r, "/usuarios", http.StatusSeeOther)
	})
}

func handleDeleteUsuario(app *App) http.HandlerFunc {
	return adminOnly(app, func(w http.ResponseWriter, r *http.Request) {
		id := parseInt(r.PathValue("id"))
		if id == currentUser(r).ID {
			http.Redirect(w, r, "/usuarios", http.StatusSeeOther)
			return
		}
		dbDeleteUser(app.db, id)
		http.Redirect(w, r, "/usuarios", http.StatusSeeOther)
	})
}
