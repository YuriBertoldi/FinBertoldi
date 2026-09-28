package handler

import (
	"database/sql"
	"fmt"
	"net/http"
	"strings"
	"time"

	"fincontrol/internal/auth"
	"fincontrol/internal/models"
	"fincontrol/internal/store"

	"github.com/xuri/excelize/v2"
)

// --- Cabeçalhos padrão por aba ---

var xlsHeaders = map[string][]string{
	"Despesas Fixas":  {"Nome", "Valor", "Categoria", "Ativa (S/N)"},
	"Parcelamentos":   {"Descrição", "Cartão", "Valor Parcela", "Parcela Atual", "Total Parcelas", "Data Início (AAAA-MM-DD)"},
	"Receitas":        {"Descrição", "Valor", "Data (AAAA-MM-DD)", "Tipo", "Recorrente (S/N)"},
	"Investimentos":   {"Instituição", "Tipo", "Valor", "Data (AAAA-MM-DD)", "Notas"},
	"Empréstimos":     {"Pessoa", "Valor", "Direção (devo/me_devem)", "Data (AAAA-MM-DD)", "Notas"},
}

var xlsSheets = []string{"Despesas Fixas", "Parcelamentos", "Receitas", "Investimentos", "Empréstimos"}

func newXlsx() *excelize.File {
	f := excelize.NewFile()

	// Estilo do cabeçalho
	headerStyle, _ := f.NewStyle(&excelize.Style{
		Font:      &excelize.Font{Bold: true, Color: "FFFFFF"},
		Fill:      excelize.Fill{Type: "pattern", Color: []string{"1A2332"}, Pattern: 1},
		Alignment: &excelize.Alignment{Horizontal: "center"},
	})

	for _, sheet := range xlsSheets {
		f.NewSheet(sheet)
		headers := xlsHeaders[sheet]
		for i, h := range headers {
			cell, _ := excelize.CoordinatesToCellName(i+1, 1)
			f.SetCellValue(sheet, cell, h)
			f.SetCellStyle(sheet, cell, cell, headerStyle)
			f.SetColWidth(sheet, colName(i+1), colName(i+1), 22)
		}
	}
	f.DeleteSheet("Sheet1")
	return f
}

func colName(n int) string {
	name, _ := excelize.ColumnNumberToName(n)
	return name
}

func boolStr(b bool) string {
	if b {
		return "S"
	}
	return "N"
}

func parseBool(s string) bool {
	s = strings.TrimSpace(strings.ToUpper(s))
	return s == "S" || s == "SIM" || s == "TRUE" || s == "1"
}

// HandleExportTemplate gera XLSX vazio com cabeçalhos + linha de exemplo
func HandleExportTemplate(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		f := newXlsx()

		// Linhas de exemplo
		f.SetCellValue("Despesas Fixas", "A2", "Aluguel")
		f.SetCellValue("Despesas Fixas", "B2", 1500.00)
		f.SetCellValue("Despesas Fixas", "C2", "Moradia")
		f.SetCellValue("Despesas Fixas", "D2", "S")

		f.SetCellValue("Parcelamentos", "A2", "TV Samsung")
		f.SetCellValue("Parcelamentos", "B2", "Nubank")
		f.SetCellValue("Parcelamentos", "C2", 250.00)
		f.SetCellValue("Parcelamentos", "D2", 1)
		f.SetCellValue("Parcelamentos", "E2", 12)
		f.SetCellValue("Parcelamentos", "F2", time.Now().Format("2006-01-02"))

		f.SetCellValue("Receitas", "A2", "Salário")
		f.SetCellValue("Receitas", "B2", 5000.00)
		f.SetCellValue("Receitas", "C2", time.Now().Format("2006-01-02"))
		f.SetCellValue("Receitas", "D2", "Salário")
		f.SetCellValue("Receitas", "E2", "S")

		f.SetCellValue("Investimentos", "A2", "Nubank")
		f.SetCellValue("Investimentos", "B2", "CDB")
		f.SetCellValue("Investimentos", "C2", 10000.00)
		f.SetCellValue("Investimentos", "D2", time.Now().Format("2006-01-02"))
		f.SetCellValue("Investimentos", "E2", "")

		f.SetCellValue("Empréstimos", "A2", "João")
		f.SetCellValue("Empréstimos", "B2", 500.00)
		f.SetCellValue("Empréstimos", "C2", "me_devem")
		f.SetCellValue("Empréstimos", "D2", time.Now().Format("2006-01-02"))
		f.SetCellValue("Empréstimos", "E2", "")

		w.Header().Set("Content-Type", "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet")
		w.Header().Set("Content-Disposition", "attachment; filename=\"modelo_finbertoldi.xlsx\"")
		f.Write(w)
	}
}

// HandleExport exporta todos os dados da família em XLSX
func HandleExport(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		fid := auth.CurrentUser(r).FamilyID
		f := newXlsx()

		// Despesas Fixas
		despesas, _ := store.GetDespesasFixas(db, fid)
		for i, d := range despesas {
			row := i + 2
			f.SetCellValue("Despesas Fixas", fmt.Sprintf("A%d", row), d.Nome)
			f.SetCellValue("Despesas Fixas", fmt.Sprintf("B%d", row), d.Valor)
			f.SetCellValue("Despesas Fixas", fmt.Sprintf("C%d", row), d.Categoria)
			f.SetCellValue("Despesas Fixas", fmt.Sprintf("D%d", row), boolStr(d.Ativa))
		}

		// Parcelamentos
		parcs, _ := store.GetParcelamentos(db, fid, false)
		for i, p := range parcs {
			row := i + 2
			f.SetCellValue("Parcelamentos", fmt.Sprintf("A%d", row), p.Descricao)
			f.SetCellValue("Parcelamentos", fmt.Sprintf("B%d", row), p.Cartao)
			f.SetCellValue("Parcelamentos", fmt.Sprintf("C%d", row), p.ValorParcela)
			f.SetCellValue("Parcelamentos", fmt.Sprintf("D%d", row), p.ParcelaAtual)
			f.SetCellValue("Parcelamentos", fmt.Sprintf("E%d", row), p.TotalParcelas)
			f.SetCellValue("Parcelamentos", fmt.Sprintf("F%d", row), p.DataInicio.Format("2006-01-02"))
		}

		// Receitas
		receitas, _ := store.GetReceitas(db, fid)
		for i, rc := range receitas {
			row := i + 2
			f.SetCellValue("Receitas", fmt.Sprintf("A%d", row), rc.Descricao)
			f.SetCellValue("Receitas", fmt.Sprintf("B%d", row), rc.Valor)
			f.SetCellValue("Receitas", fmt.Sprintf("C%d", row), rc.Data.Format("2006-01-02"))
			f.SetCellValue("Receitas", fmt.Sprintf("D%d", row), rc.Tipo)
			f.SetCellValue("Receitas", fmt.Sprintf("E%d", row), boolStr(rc.Recorrente))
		}

		// Investimentos
		invs, _ := store.GetInvestimentos(db, fid)
		for i, inv := range invs {
			row := i + 2
			f.SetCellValue("Investimentos", fmt.Sprintf("A%d", row), inv.Instituicao)
			f.SetCellValue("Investimentos", fmt.Sprintf("B%d", row), inv.Tipo)
			f.SetCellValue("Investimentos", fmt.Sprintf("C%d", row), inv.Valor)
			f.SetCellValue("Investimentos", fmt.Sprintf("D%d", row), inv.Data.Format("2006-01-02"))
			f.SetCellValue("Investimentos", fmt.Sprintf("E%d", row), inv.Notas)
		}

		// Empréstimos
		emps, _ := store.GetEmprestimos(db, fid)
		for i, e := range emps {
			row := i + 2
			f.SetCellValue("Empréstimos", fmt.Sprintf("A%d", row), e.Pessoa)
			f.SetCellValue("Empréstimos", fmt.Sprintf("B%d", row), e.Valor)
			f.SetCellValue("Empréstimos", fmt.Sprintf("C%d", row), e.Direcao)
			f.SetCellValue("Empréstimos", fmt.Sprintf("D%d", row), e.Data.Format("2006-01-02"))
			f.SetCellValue("Empréstimos", fmt.Sprintf("E%d", row), e.Notas)
		}

		nome := fmt.Sprintf("finbertoldi_%s.xlsx", time.Now().Format("2006-01-02"))
		w.Header().Set("Content-Type", "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet")
		w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s\"", nome))
		f.Write(w)
	}
}

// HandleImportPage renderiza a página de importação
func HandleImportPage(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		render(w, "importexport", models.ImportExportPage{
			BasePage: bp(db, r, "importexport", "Importar / Exportar"),
		})
	}
}

// HandleImport processa upload de XLSX e insere os dados
func HandleImport(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		fid := auth.CurrentUser(r).FamilyID

		r.ParseMultipartForm(10 << 20) // 10 MB
		file, _, err := r.FormFile("arquivo")
		if err != nil {
			render(w, "importexport", models.ImportExportPage{
				BasePage: bp(db, r, "importexport", "Importar / Exportar"),
				Erro:     "Nenhum arquivo enviado.",
			})
			return
		}
		defer file.Close()

		f, err := excelize.OpenReader(file)
		if err != nil {
			render(w, "importexport", models.ImportExportPage{
				BasePage: bp(db, r, "importexport", "Importar / Exportar"),
				Erro:     "Arquivo inválido ou corrompido.",
			})
			return
		}
		defer f.Close()

		resultado := models.ImportResultado{}

		// Despesas Fixas
		if rows, err := f.GetRows("Despesas Fixas"); err == nil {
			for _, row := range rows[1:] {
				if len(row) < 1 || strings.TrimSpace(row[0]) == "" {
					continue
				}
				nome := str(row, 0)
				valor := parseFloat(str(row, 1))
				cat := str(row, 2)
				ativa := !parseBool(str(row, 3)) || parseBool(str(row, 3)) // sempre true na importação
				_ = ativa
				if err := store.CreateDespesa(db, fid, nome, valor, cat); err == nil {
					resultado.Despesas++
				}
			}
		}

		// Parcelamentos
		if rows, err := f.GetRows("Parcelamentos"); err == nil {
			for _, row := range rows[1:] {
				if len(row) < 1 || strings.TrimSpace(row[0]) == "" {
					continue
				}
				descricao := str(row, 0)
				cartao := str(row, 1)
				valorParc := parseFloat(str(row, 2))
				parcelaAtual := parseInt(str(row, 3))
				totalParcelas := parseInt(str(row, 4))
				dataInicio := parseDate(str(row, 5))
				if parcelaAtual == 0 {
					parcelaAtual = 1
				}
				if totalParcelas == 0 {
					totalParcelas = 1
				}
				if err := store.CreateParcelamento(db, fid, descricao, cartao, valorParc, parcelaAtual, totalParcelas, dataInicio, false, 0, 0); err == nil {
					resultado.Parcelamentos++
				}
			}
		}

		// Receitas
		if rows, err := f.GetRows("Receitas"); err == nil {
			for _, row := range rows[1:] {
				if len(row) < 1 || strings.TrimSpace(row[0]) == "" {
					continue
				}
				descricao := str(row, 0)
				valor := parseFloat(str(row, 1))
				data := parseDate(str(row, 2))
				tipo := str(row, 3)
				recorrente := parseBool(str(row, 4))
				if err := store.CreateReceita(db, fid, descricao, valor, data, tipo, recorrente); err == nil {
					resultado.Receitas++
				}
			}
		}

		// Investimentos
		if rows, err := f.GetRows("Investimentos"); err == nil {
			for _, row := range rows[1:] {
				if len(row) < 1 || strings.TrimSpace(row[0]) == "" {
					continue
				}
				inst := str(row, 0)
				tipo := str(row, 1)
				valor := parseFloat(str(row, 2))
				data := parseDate(str(row, 3))
				notas := str(row, 4)
				if err := store.CreateInvestimento(db, fid, inst, tipo, valor, data, notas); err == nil {
					resultado.Investimentos++
				}
			}
		}

		// Empréstimos
		if rows, err := f.GetRows("Empréstimos"); err == nil {
			for _, row := range rows[1:] {
				if len(row) < 1 || strings.TrimSpace(row[0]) == "" {
					continue
				}
				pessoa := str(row, 0)
				valor := parseFloat(str(row, 1))
				direcao := str(row, 2)
				if direcao != "devo" && direcao != "me_devem" {
					direcao = "me_devem"
				}
				data := parseDate(str(row, 3))
				notas := str(row, 4)
				if err := store.CreateEmprestimo(db, fid, pessoa, valor, direcao, data, notas); err == nil {
					resultado.Emprestimos++
				}
			}
		}

		render(w, "importexport", models.ImportExportPage{
			BasePage:  bp(db, r, "importexport", "Importar / Exportar"),
			Resultado: &resultado,
		})
	}
}

func str(row []string, i int) string {
	if i < len(row) {
		return strings.TrimSpace(row[i])
	}
	return ""
}
