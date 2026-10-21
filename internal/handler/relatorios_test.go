package handler

import (
	"io"
	"os"
	"testing"

	"fincontrol/internal/models"
)

// Erro de tipo dentro do template (ex.: comparar texto com numero) so aparece
// na execucao: o relatorio sai cortado no meio e o erro vai so para o log.
// Foi o que aconteceu com o card "Sobra do Mes" do relatorio geral. Aqui cada
// relatorio e executado com um item em cada lista, para os range rodarem.
func TestRelatorios_ExecutamSemErro(t *testing.T) {
	pastaDoTeste, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir("../.."); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { os.Chdir(pastaDoTeste) })
	InitReportTemplates()

	relatorios := map[string]any{
		"despesas": rDespesas{Despesas: []models.DespesaFixa{{}}, Parcelamentos: []models.Parcelamento{{}}},
		"receitas": rReceitas{Receitas: []models.Receita{{}}},
		"investimentos": rInvestimentos{Investimentos: []models.Investimento{{}}, Totais: map[string]float64{"CDB": 1},
			HistoricoReserva: []models.ReservaEM{{}}},
		"emprestimos": rEmprestimos{Emprestimos: []models.Emprestimo{{}}},
		"dashboard": rDashboard{Sobra: "-R$ 10,00", Sobra_f: -10,
			DespesasBasicas: []models.DespesaMes{{}}, DespesasCartao: []models.DespesaMes{{}}, DespesasVR: []models.DespesaMes{{}},
			Parcelamentos: []models.Parcelamento{{}}, Historico: []models.MesResumo{{}},
			Projecao: models.MontarProjecao([]models.MesProjetado{{Receitas: 100, DespesasFixas: 50}})},
	}
	for nome, dados := range relatorios {
		t.Run(nome, func(t *testing.T) {
			if err := tmplReports[nome].ExecuteTemplate(io.Discard, "relatorio-base", dados); err != nil {
				t.Errorf("relatorio %s: %v", nome, err)
			}
		})
	}
}
