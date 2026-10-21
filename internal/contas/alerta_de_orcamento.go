package contas

import (
	"database/sql"
	"fmt"
	"log"
	"time"

	"fincontrol/internal/models"
	"fincontrol/internal/store"
)

// alertaDeOrcamento e um aviso a enviar: a categoria e a faixa atingida
// ("alerta" a partir de 80%, "estourado" acima de 100%).
type alertaDeOrcamento struct {
	Orcamento models.OrcamentoDaCategoria
	Faixa     string
}

// alertasNovos escolhe o que ainda nao foi avisado no mes. Cada faixa sai uma
// unica vez por categoria; quem ja estourou nao recebe mais o aviso de 80%
// (nem quando sobe direto para acima de 100%, nem se o limite for aumentado
// depois e a categoria voltar para a faixa de alerta).
func alertasNovos(orcamentos []models.OrcamentoDaCategoria, jaEnviados map[int]map[string]bool) []alertaDeOrcamento {
	var novos []alertaDeOrcamento
	for _, orcamento := range orcamentos {
		enviados := jaEnviados[orcamento.CategoriaID]
		switch orcamento.Situacao() {
		case "estourado":
			if !enviados["estourado"] {
				novos = append(novos, alertaDeOrcamento{Orcamento: orcamento, Faixa: "estourado"})
			}
		case "alerta":
			if !enviados["alerta"] && !enviados["estourado"] {
				novos = append(novos, alertaDeOrcamento{Orcamento: orcamento, Faixa: "alerta"})
			}
		}
	}
	return novos
}

// textoDoAlerta usa o formato do WhatsApp (* = negrito), o mesmo que o envio
// mensal converte para e-mail e Telegram.
func textoDoAlerta(alerta alertaDeOrcamento, mes time.Time) string {
	orcamento := alerta.Orcamento
	titulo := fmt.Sprintf("*Orçamento de %s em %.0f%%*", orcamento.Categoria, orcamento.PercentualUsado())
	if alerta.Faixa == "estourado" {
		titulo = fmt.Sprintf("*Orçamento de %s estourado: %.0f%%*", orcamento.Categoria, orcamento.PercentualUsado())
	}
	return fmt.Sprintf("%s\nGasto em %s: R$ %s de R$ %s.\nDespesas fixas R$ %s · extrato R$ %s",
		titulo, MesDisplay(mes), waValor(orcamento.Gasto()), waValor(orcamento.Limite),
		waValor(orcamento.GastoEmDespesasFixas), waValor(orcamento.GastoEmTransacoes))
}

// EnviarAlertasDeOrcamento avisa, a partir das 8h, cada usuario que pediu o
// alerta sobre as categorias da familia dele que chegaram a 80% ou passaram de
// 100% no mes. O registro e gravado mesmo se o envio falhar, como no envio
// mensal: sem isso o mesmo erro se repetiria a cada hora.
func EnviarAlertasDeOrcamento(db *sql.DB, agora time.Time) {
	if agora.Hour() < 8 {
		return
	}
	mes := time.Date(agora.Year(), agora.Month(), 1, 0, 0, 0, 0, time.UTC)
	usuarios, err := store.UsuariosComAlertaDeOrcamento(db)
	if err != nil {
		log.Printf("[alerta-orcamento] buscar usuarios: %v", err)
		return
	}

	orcamentosPorFamilia := map[int][]models.OrcamentoDaCategoria{}
	for _, usuario := range usuarios {
		orcamentos, calculado := orcamentosPorFamilia[usuario.User.FamilyID]
		if !calculado {
			orcamentos, err = store.OrcamentosDoMes(db, usuario.User.FamilyID, mes)
			if err != nil {
				log.Printf("[alerta-orcamento] orcamentos da familia %d: %v", usuario.User.FamilyID, err)
				continue
			}
			orcamentosPorFamilia[usuario.User.FamilyID] = orcamentos
		}
		jaEnviados, err := store.AlertasDeOrcamentoEnviados(db, usuario.User.ID, mes)
		if err != nil {
			log.Printf("[alerta-orcamento] alertas enviados ao usuario %d: %v", usuario.User.ID, err)
			continue
		}

		for _, alerta := range alertasNovos(orcamentos, jaEnviados) {
			assunto := fmt.Sprintf("Orçamento de %s - FinBertoldi", alerta.Orcamento.Categoria)
			resultados := enviarMensagem(db, &usuario.User, usuario.Notificacoes, assunto, textoDoAlerta(alerta, mes),
				usuario.Notificacoes.AlertaOrcamentoEmail, usuario.Notificacoes.AlertaOrcamentoTelegram)
			for _, resultado := range resultados {
				if resultado.Erro != "" {
					log.Printf("[alerta-orcamento] user=%d %s: %s", usuario.User.ID, resultado.Canal, resultado.Erro)
				}
			}
			faixasParaRegistrar := []string{alerta.Faixa}
			if alerta.Faixa == "estourado" {
				faixasParaRegistrar = append(faixasParaRegistrar, "alerta")
			}
			for _, faixa := range faixasParaRegistrar {
				if err := store.RegistrarAlertaDeOrcamentoEnviado(db, usuario.User.ID, alerta.Orcamento.CategoriaID, mes, faixa); err != nil {
					log.Printf("[alerta-orcamento] registrar envio: %v", err)
				}
			}
		}
	}
}
