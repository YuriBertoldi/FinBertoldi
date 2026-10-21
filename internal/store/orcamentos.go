package store

import (
	"database/sql"
	"time"

	"fincontrol/internal/models"
)

// DefinirOrcamentoMensalDaCategoria grava o limite mensal da categoria;
// nil remove o orcamento.
func DefinirOrcamentoMensalDaCategoria(db *sql.DB, fid, categoriaID int, limite *float64) error {
	_, err := db.Exec(`UPDATE categorias SET orcamento_mensal=$1 WHERE id=$2 AND family_id=$3`,
		limite, categoriaID, fid)
	return err
}

// OrcamentosDoMes devolve, para cada categoria ativa com orcamento, quanto ja
// foi gasto no mes. O gasto soma duas fontes:
//   - despesas fixas ativas da categoria (valor mensal);
//   - debitos do extrato no mes, na categoria. Transacoes "convertida" ficam
//     de fora porque ja viraram (ou foram vinculadas a) uma despesa, que entra
//     na primeira soma; "ignorada" fica de fora por definicao.
func OrcamentosDoMes(db *sql.DB, fid int, mes time.Time) ([]models.OrcamentoDaCategoria, error) {
	linhas, err := db.Query(`
		SELECT c.id, c.nome, c.cor, c.orcamento_mensal,
		       COALESCE((SELECT SUM(df.valor) FROM despesas_fixas df
		                 WHERE df.family_id = c.family_id AND df.ativa AND df.categoria = c.nome), 0),
		       COALESCE((SELECT SUM(ABS(t.valor)) FROM transacoes_banco t
		                 WHERE t.family_id = c.family_id AND t.tipo = 'debito' AND t.categoria = c.nome
		                   AND t.status NOT IN ('convertida', 'ignorada')
		                   AND date_trunc('month', t.data) = date_trunc('month', $2::date)), 0)
		FROM categorias c
		WHERE c.family_id = $1 AND c.ativo AND c.orcamento_mensal IS NOT NULL
		ORDER BY c.nome`, fid, mes)
	if err != nil {
		return nil, err
	}
	defer linhas.Close()

	var orcamentos []models.OrcamentoDaCategoria
	for linhas.Next() {
		var orcamento models.OrcamentoDaCategoria
		if err := linhas.Scan(&orcamento.CategoriaID, &orcamento.Categoria, &orcamento.Cor, &orcamento.Limite,
			&orcamento.GastoEmDespesasFixas, &orcamento.GastoEmTransacoes); err != nil {
			return nil, err
		}
		orcamentos = append(orcamentos, orcamento)
	}
	return orcamentos, linhas.Err()
}
