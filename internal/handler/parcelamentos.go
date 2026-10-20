package handler

import (
	"database/sql"
	"fincontrol/internal/auth"
	"fincontrol/internal/store"
	"fmt"
	"log"
	"net/http"
)

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
