package handler

import (
	"encoding/json"
	"fmt"
	"hash/crc32"
	"html/template"
	"log"
	"net/http"
	"os"
	"strconv"
	"time"
)

var tmplPages = map[string]*template.Template{}

// assetVersion muda sempre que static/app.css muda, para o navegador não usar CSS antigo do cache.
var assetVersion = func() string {
	b, err := os.ReadFile("static/app.css")
	if err != nil {
		return strconv.FormatInt(time.Now().Unix(), 36)
	}
	return strconv.FormatUint(uint64(crc32.ChecksumIEEE(b)), 36)
}()

func buildFuncMap() template.FuncMap {
	return template.FuncMap{
		"assetV": func() string { return assetVersion },
		// dict monta um mapa para passar mais de um valor a um template: dict "K" x "P" $
		"dict": func(kv ...any) map[string]any {
			m := map[string]any{}
			for i := 0; i+1 < len(kv); i += 2 {
				if k, ok := kv[i].(string); ok {
					m[k] = kv[i+1]
				}
			}
			return m
		},
		"brl": func(v float64) template.HTML {
			return template.HTML(`<span class="money">` + formatBRL(v) + `</span>`)
		},
		"date":      func(t time.Time) string { return t.Format("02/01/2006") },
		"dateInput": func(t time.Time) string { return t.Format("2006-01-02") },
		"neg":       func(v float64) bool { return v < 0 },
		"abs": func(v float64) float64 {
			if v < 0 {
				return -v
			}
			return v
		},
		"progress": func(atual, total int) string { return fmt.Sprintf("%dx%d", atual, total) },
		"sub":      func(a, b float64) float64 { return a - b },
		"pct": func(v, total float64) string {
			if total == 0 {
				return "0%"
			}
			return fmt.Sprintf("%.0f%%", (v/total)*100)
		},
		"mesDisplay": mesDisplay,
		"seq": func(n int) []int {
			s := make([]int, n)
			for i := range s {
				s[i] = i + 1
			}
			return s
		},
		"json": func(v any) template.JS {
			b, _ := json.Marshal(v)
			return template.JS(b)
		},
		"deref": func(p *float64) float64 {
			if p == nil {
				return 0
			}
			return *p
		},
		"contains": func(slice []string, s string) bool {
			for _, v := range slice {
				if v == s {
					return true
				}
			}
			return false
		},
	}
}

// InitTemplates carrega e compila todos os templates.
func InitTemplates() {
	pages := []string{"dashboard", "despesas", "pagamentos", "receitas", "investimentos", "planejamento", "emprestimos", "usuarios", "familias", "cadastros", "minha-familia", "importexport", "transacoes", "integracoes", "notificacoes"}
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
	reg := template.Must(template.New("register").Funcs(buildFuncMap()).ParseFiles("templates/register.html"))
	tmplPages["register"] = reg
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
