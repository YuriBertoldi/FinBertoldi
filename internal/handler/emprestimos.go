package handler

import (
	"database/sql"
	"fincontrol/internal/auth"
	"fincontrol/internal/models"
	"fincontrol/internal/store"
	"log"
	"net/http"
)

func HandleEmprestimos(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		fid := auth.CurrentUser(r).FamilyID
		list, _ := store.GetEmprestimos(db, fid)
		var totalDevo, totalAReceber float64
		for _, e := range list {
			if e.Pago {
				continue
			}
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
