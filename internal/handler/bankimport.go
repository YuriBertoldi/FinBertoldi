package handler

import (
	"database/sql"
	"encoding/csv"
	"fmt"
	"io"
	"math"
	"net/http"
	"strconv"
	"strings"
	"time"
	"unicode"

	"fincontrol/internal/auth"
	"fincontrol/internal/models"
	"fincontrol/internal/store"

	"github.com/aclindsa/ofxgo"
)

// ---------------------------------------------------------------------------
// CSV Parser
// ---------------------------------------------------------------------------

// csvColumnMap maps normalized header names to standard field names.
var csvColumnMap = map[string]string{
	"data":        "data",
	"date":        "data",
	"fecha":       "data",
	"descricao":   "descricao",
	"descrição":   "descricao",
	"description": "descricao",
	"titulo":      "descricao",
	"título":      "descricao",
	"valor":       "valor",
	"value":       "valor",
	"amount":      "valor",
	"categoria":   "categoria",
	"category":    "categoria",
}

func normalizeHeader(s string) string {
	s = strings.TrimSpace(s)
	s = strings.ToLower(s)
	// Remove BOM
	s = strings.TrimLeft(s, "\xef\xbb\xbf")
	// Strip non-printable
	return strings.Map(func(r rune) rune {
		if unicode.IsPrint(r) {
			return r
		}
		return -1
	}, s)
}

func parseCSVFile(r io.Reader) ([]models.TransacaoBanco, error) {
	// Read all content so we can retry with different separator
	data, err := io.ReadAll(r)
	if err != nil {
		return nil, fmt.Errorf("erro ao ler arquivo CSV: %w", err)
	}

	// Detect separator: try comma first, then semicolon
	sep := ','
	firstLine := string(data)
	if idx := strings.Index(firstLine, "\n"); idx > 0 {
		firstLine = firstLine[:idx]
	}
	if strings.Count(firstLine, ";") > strings.Count(firstLine, ",") {
		sep = ';'
	}

	reader := csv.NewReader(strings.NewReader(string(data)))
	reader.Comma = sep
	reader.LazyQuotes = true
	reader.TrimLeadingSpace = true

	header, err := reader.Read()
	if err != nil {
		return nil, fmt.Errorf("erro ao ler cabeçalho CSV: %w", err)
	}

	// Map columns
	colIdx := map[string]int{}
	for i, h := range header {
		norm := normalizeHeader(h)
		if field, ok := csvColumnMap[norm]; ok {
			colIdx[field] = i
		}
	}

	dataIdx, hasData := colIdx["data"]
	descIdx, hasDesc := colIdx["descricao"]
	valorIdx, hasValor := colIdx["valor"]
	if !hasData || !hasDesc || !hasValor {
		return nil, fmt.Errorf("CSV deve conter colunas: data, descricao, valor (encontradas: %v)", header)
	}
	catIdx, hasCat := colIdx["categoria"]

	var txns []models.TransacaoBanco
	lineNum := 1
	for {
		record, err := reader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			// Try semicolon split for mixed formats
			continue
		}
		lineNum++

		if dataIdx >= len(record) || descIdx >= len(record) || valorIdx >= len(record) {
			continue
		}

		data, err := parseDateFlex(strings.TrimSpace(record[dataIdx]))
		if err != nil {
			continue
		}

		desc := strings.TrimSpace(record[descIdx])
		if desc == "" {
			continue
		}

		valor, err := parseValorBR(strings.TrimSpace(record[valorIdx]))
		if err != nil {
			continue
		}

		tipo := "debito"
		if valor > 0 {
			tipo = "credito"
		}

		cat := ""
		if hasCat && catIdx < len(record) {
			cat = strings.TrimSpace(record[catIdx])
		}

		txns = append(txns, models.TransacaoBanco{
			Data:      data,
			Descricao: desc,
			Valor:     valor,
			Tipo:      tipo,
			Categoria: cat,
			Origem:    "csv",
			FitID:     fmt.Sprintf("csv-%s-%s-%.2f", data.Format("2006-01-02"), desc, valor),
		})
	}

	if len(txns) == 0 {
		return nil, fmt.Errorf("nenhuma transação válida encontrada no CSV")
	}
	return txns, nil
}

// parseDateFlex handles DD/MM/YYYY, YYYY-MM-DD, DD-MM-YYYY
func parseDateFlex(s string) (time.Time, error) {
	formats := []string{
		"02/01/2006",
		"2006-01-02",
		"02-01-2006",
		"2/1/2006",
		"01/02/2006", // US format fallback
	}
	for _, f := range formats {
		if t, err := time.Parse(f, s); err == nil {
			return t, nil
		}
	}
	return time.Time{}, fmt.Errorf("formato de data não reconhecido: %s", s)
}

// parseValorBR handles Brazilian (1.234,56) and international (1,234.56) number formats
func parseValorBR(s string) (float64, error) {
	s = strings.TrimSpace(s)
	// Remove currency symbols
	s = strings.ReplaceAll(s, "R$", "")
	s = strings.ReplaceAll(s, "R$ ", "")
	s = strings.TrimSpace(s)
	if s == "" {
		return 0, fmt.Errorf("valor vazio")
	}

	// Detect format by last separator
	lastComma := strings.LastIndex(s, ",")
	lastDot := strings.LastIndex(s, ".")

	if lastComma > lastDot {
		// BR format: 1.234,56 → remove dots, replace comma with dot
		s = strings.ReplaceAll(s, ".", "")
		s = strings.Replace(s, ",", ".", 1)
	} else if lastDot > lastComma {
		// Int format: 1,234.56 → remove commas
		s = strings.ReplaceAll(s, ",", "")
	}

	return strconv.ParseFloat(s, 64)
}

// ---------------------------------------------------------------------------
// OFX Parser
// ---------------------------------------------------------------------------

func parseOFXFile(r io.Reader) ([]models.TransacaoBanco, string, error) {
	resp, err := ofxgo.ParseResponse(r)
	if err != nil {
		return nil, "", fmt.Errorf("erro ao parsear OFX: %w", err)
	}

	banco := ""
	if resp.Signon.Fid != "" {
		banco = string(resp.Signon.Fid)
	}
	if resp.Signon.Org != "" {
		if banco != "" {
			banco = string(resp.Signon.Org) + " - " + banco
		} else {
			banco = string(resp.Signon.Org)
		}
	}

	var txns []models.TransacaoBanco

	for _, msg := range resp.Bank {
		if stmt, ok := msg.(*ofxgo.StatementResponse); ok {
			for _, t := range stmt.BankTranList.Transactions {
				txn := ofxTxnToModel(t, banco)
				txns = append(txns, txn)
			}
		}
	}

	for _, msg := range resp.CreditCard {
		if stmt, ok := msg.(*ofxgo.CCStatementResponse); ok {
			for _, t := range stmt.BankTranList.Transactions {
				txn := ofxTxnToModel(t, banco)
				txns = append(txns, txn)
			}
		}
	}

	if len(txns) == 0 {
		return nil, banco, fmt.Errorf("nenhuma transação encontrada no arquivo OFX")
	}
	return txns, banco, nil
}

func ofxTxnToModel(t ofxgo.Transaction, banco string) models.TransacaoBanco {
	valor, _ := t.TrnAmt.Float64()
	desc := string(t.Name)
	if desc == "" {
		desc = string(t.Memo)
	}

	tipo := "debito"
	if valor > 0 {
		tipo = "credito"
	}

	return models.TransacaoBanco{
		Data:      t.DtPosted.Time,
		Descricao: desc,
		Valor:     valor,
		Tipo:      tipo,
		Origem:    "ofx",
		Banco:     banco,
		FitID:     string(t.FiTID),
	}
}

// ---------------------------------------------------------------------------
// Handlers
// ---------------------------------------------------------------------------

func HandleTransacoes(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		u := auth.CurrentUser(r)
		fid := u.FamilyID

		mes := r.URL.Query().Get("mes")
		if mes == "" {
			mes = time.Now().Format("2006-01")
		}
		origem := r.URL.Query().Get("origem")
		status := r.URL.Query().Get("status")
		banco := r.URL.Query().Get("banco")

		txns, _ := store.GetTransacoesBanco(db, fid, mes, origem, status, banco)
		cats, _ := store.GetCategorias(db, fid)
		contas, _ := store.GetPluggyItems(db, fid)

		var totalEnt, totalSai float64
		for _, t := range txns {
			if t.Valor > 0 {
				totalEnt += t.Valor
			} else {
				totalSai += math.Abs(t.Valor)
			}
		}

		render(w, "transacoes", models.TransacoesPage{
			BasePage: models.BasePage{
				CurrentUser:    u,
				Active:         "transacoes",
				Title:          "Transações Bancárias",
				BlockedScreens: u.BlockedScreens,
			},
			Transacoes:    txns,
			Categorias:    cats,
			Contas:        contas,
			TotalEntradas: totalEnt,
			TotalSaidas:   totalSai,
			Filtros: models.TransacoesFiltros{
				Mes:    mes,
				Origem: origem,
				Status: status,
				Banco:  banco,
			},
		})
	}
}

func HandleImportCSV(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		u := auth.CurrentUser(r)
		fid := u.FamilyID

		file, _, err := r.FormFile("arquivo")
		if err != nil {
			renderTransacoesComErro(w, r, db, u, "Selecione um arquivo CSV.")
			return
		}
		defer file.Close()

		banco := r.FormValue("banco")
		txns, err := parseCSVFile(file)
		if err != nil {
			renderTransacoesComErro(w, r, db, u, err.Error())
			return
		}

		if banco != "" {
			for i := range txns {
				txns[i].Banco = banco
			}
		}

		novos, err := store.CreateTransacaoBancoBatch(db, fid, txns)
		if err != nil {
			renderTransacoesComErro(w, r, db, u, "Erro ao salvar: "+err.Error())
			return
		}

		renderTransacoesComResultado(w, r, db, u, &models.ImportBancoResultado{
			Total:      len(txns),
			Novos:      novos,
			Duplicados: len(txns) - novos,
		})
	}
}

func HandleImportOFX(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		u := auth.CurrentUser(r)
		fid := u.FamilyID

		file, _, err := r.FormFile("arquivo")
		if err != nil {
			renderTransacoesComErro(w, r, db, u, "Selecione um arquivo OFX.")
			return
		}
		defer file.Close()

		txns, banco, err := parseOFXFile(file)
		if err != nil {
			renderTransacoesComErro(w, r, db, u, err.Error())
			return
		}

		// Allow manual bank name override
		if b := r.FormValue("banco"); b != "" {
			banco = b
			for i := range txns {
				txns[i].Banco = banco
			}
		}

		novos, err := store.CreateTransacaoBancoBatch(db, fid, txns)
		if err != nil {
			renderTransacoesComErro(w, r, db, u, "Erro ao salvar: "+err.Error())
			return
		}

		renderTransacoesComResultado(w, r, db, u, &models.ImportBancoResultado{
			Total:      len(txns),
			Novos:      novos,
			Duplicados: len(txns) - novos,
		})
	}
}

func HandleTransacaoConverter(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		u := auth.CurrentUser(r)
		fid := u.FamilyID
		id := parseInt(r.PathValue("id"))
		destino := r.FormValue("destino") // "despesa" ou "receita"
		categoria := r.FormValue("categoria")
		tipo := r.FormValue("tipo")

		var err error
		if destino == "despesa" {
			err = store.ConverterTransacaoEmDespesa(db, fid, id, categoria)
		} else {
			err = store.ConverterTransacaoEmReceita(db, fid, id, tipo)
		}

		if err != nil {
			renderTransacoesComErro(w, r, db, u, "Erro ao converter: "+err.Error())
			return
		}
		http.Redirect(w, r, "/transacoes", http.StatusSeeOther)
	}
}

func HandleTransacaoIgnorar(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		u := auth.CurrentUser(r)
		id := parseInt(r.PathValue("id"))
		store.IgnorarTransacao(db, u.FamilyID, id)
		http.Redirect(w, r, "/transacoes", http.StatusSeeOther)
	}
}

func HandleTransacaoCategorizar(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		u := auth.CurrentUser(r)
		id := parseInt(r.PathValue("id"))
		categoria := r.FormValue("categoria")
		store.CategorizarTransacao(db, u.FamilyID, id, categoria)
		http.Redirect(w, r, "/transacoes", http.StatusSeeOther)
	}
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

func renderTransacoesComErro(w http.ResponseWriter, r *http.Request, db *sql.DB, u *models.User, erro string) {
	fid := u.FamilyID
	mes := time.Now().Format("2006-01")
	txns, _ := store.GetTransacoesBanco(db, fid, mes, "", "", "")
	cats, _ := store.GetCategorias(db, fid)
	contas, _ := store.GetPluggyItems(db, fid)

	render(w, "transacoes", models.TransacoesPage{
		BasePage: models.BasePage{
			CurrentUser:    u,
			Active:         "transacoes",
			Title:          "Transações Bancárias",
			BlockedScreens: u.BlockedScreens,
		},
		Transacoes: txns,
		Categorias: cats,
		Contas:     contas,
		Filtros:    models.TransacoesFiltros{Mes: mes},
		Erro:       erro,
	})
}

func renderTransacoesComResultado(w http.ResponseWriter, r *http.Request, db *sql.DB, u *models.User, res *models.ImportBancoResultado) {
	fid := u.FamilyID
	mes := time.Now().Format("2006-01")
	txns, _ := store.GetTransacoesBanco(db, fid, mes, "", "", "")
	cats, _ := store.GetCategorias(db, fid)
	contas, _ := store.GetPluggyItems(db, fid)

	var totalEnt, totalSai float64
	for _, t := range txns {
		if t.Valor > 0 {
			totalEnt += t.Valor
		} else {
			totalSai += math.Abs(t.Valor)
		}
	}

	render(w, "transacoes", models.TransacoesPage{
		BasePage: models.BasePage{
			CurrentUser:    u,
			Active:         "transacoes",
			Title:          "Transações Bancárias",
			BlockedScreens: u.BlockedScreens,
		},
		Transacoes:    txns,
		Categorias:    cats,
		Contas:        contas,
		TotalEntradas: totalEnt,
		TotalSaidas:   totalSai,
		Filtros:       models.TransacoesFiltros{Mes: mes},
		Resultado:     res,
	})
}
