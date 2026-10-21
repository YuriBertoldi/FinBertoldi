package store

import (
	"database/sql"
	"sort"
	"strings"
	"unicode"

	"fincontrol/internal/models"

	"golang.org/x/text/runes"
	"golang.org/x/text/transform"
	"golang.org/x/text/unicode/norm"
)

// CriarRegraDeCategorizacao grava a regra se a categoria de destino for da familia.
func CriarRegraDeCategorizacao(db *sql.DB, fid int, campo, texto string, categoriaID int) error {
	_, err := db.Exec(`INSERT INTO regras_de_categorizacao (family_id, campo, texto, categoria_id)
		SELECT $1, $2, $3, id FROM categorias WHERE id = $4 AND family_id = $1`,
		fid, campo, strings.TrimSpace(texto), categoriaID)
	return err
}

func ExcluirRegraDeCategorizacao(db *sql.DB, fid, id int) error {
	_, err := db.Exec(`DELETE FROM regras_de_categorizacao WHERE id=$1 AND family_id=$2`, id, fid)
	return err
}

// RegrasDeCategorizacao devolve as regras na ordem em que sao aplicadas.
func RegrasDeCategorizacao(db *sql.DB, fid int) ([]models.RegraDeCategorizacao, error) {
	linhas, err := db.Query(`
		SELECT r.id, r.campo, r.texto, r.categoria_id, c.nome
		FROM regras_de_categorizacao r JOIN categorias c ON c.id = r.categoria_id
		WHERE r.family_id = $1`, fid)
	if err != nil {
		return nil, err
	}
	defer linhas.Close()

	var regras []models.RegraDeCategorizacao
	for linhas.Next() {
		var regra models.RegraDeCategorizacao
		if err := linhas.Scan(&regra.ID, &regra.Campo, &regra.Texto, &regra.CategoriaID, &regra.Categoria); err != nil {
			return nil, err
		}
		regras = append(regras, regra)
	}
	ordenarPorPrioridade(regras)
	return regras, linhas.Err()
}

// CategoriasDoBancoSemRegra lista as categorias vindas do banco em transacoes
// pendentes que nenhuma regra cobre ainda: sao as sugestoes da tela de regras.
func CategoriasDoBancoSemRegra(db *sql.DB, fid int) ([]string, error) {
	linhas, err := db.Query(`
		SELECT DISTINCT t.categoria_do_banco FROM transacoes_banco t
		WHERE t.family_id = $1 AND t.status = 'pendente' AND t.categoria_do_banco <> ''
		  AND NOT EXISTS (SELECT 1 FROM regras_de_categorizacao r
		                  WHERE r.family_id = $1 AND r.campo = 'categoria_do_banco'
		                    AND lower(r.texto) = lower(t.categoria_do_banco))
		ORDER BY 1`, fid)
	if err != nil {
		return nil, err
	}
	defer linhas.Close()
	var categorias []string
	for linhas.Next() {
		var categoria string
		if err := linhas.Scan(&categoria); err != nil {
			return nil, err
		}
		categorias = append(categorias, categoria)
	}
	return categorias, linhas.Err()
}

// AplicarRegrasDeCategorizacao categoriza as transacoes pendentes da familia
// cuja categoria ainda nao e uma das cadastradas (veio do banco ou vazia).
// Nao toca em categoria escolhida a mao (status "categorizada"), nem em
// transacoes convertidas ou ignoradas. Devolve quantas mudaram.
func AplicarRegrasDeCategorizacao(db *sql.DB, fid int) (int, error) {
	// Transacoes que entraram depois da migration (ex.: sincronizacao do
	// Pluggy, que grava direto no banco) ainda nao tem a categoria do banco
	// guardada a parte.
	if _, err := db.Exec(`UPDATE transacoes_banco t SET categoria_do_banco = t.categoria
		WHERE t.family_id = $1 AND t.categoria_do_banco = '' AND t.categoria <> ''
		  AND NOT EXISTS (SELECT 1 FROM categorias c WHERE c.family_id = t.family_id AND c.nome = t.categoria)`, fid); err != nil {
		return 0, err
	}

	regras, err := RegrasDeCategorizacao(db, fid)
	if err != nil || len(regras) == 0 {
		return 0, err
	}

	linhas, err := db.Query(`
		SELECT t.id, t.descricao, t.categoria_do_banco FROM transacoes_banco t
		WHERE t.family_id = $1 AND t.status = 'pendente'
		  AND NOT EXISTS (SELECT 1 FROM categorias c WHERE c.family_id = t.family_id AND c.nome = t.categoria)`, fid)
	if err != nil {
		return 0, err
	}
	type categorizacao struct {
		transacaoID int
		categoria   string
	}
	var categorizacoes []categorizacao
	for linhas.Next() {
		var transacaoID int
		var descricao, categoriaDoBanco string
		if err := linhas.Scan(&transacaoID, &descricao, &categoriaDoBanco); err != nil {
			linhas.Close()
			return 0, err
		}
		if categoria := CategoriaPelaPrimeiraRegra(regras, descricao, categoriaDoBanco); categoria != "" {
			categorizacoes = append(categorizacoes, categorizacao{transacaoID, categoria})
		}
	}
	linhas.Close()
	if err := linhas.Err(); err != nil {
		return 0, err
	}

	for _, c := range categorizacoes {
		if _, err := db.Exec(`UPDATE transacoes_banco SET categoria=$1 WHERE id=$2 AND family_id=$3`,
			c.categoria, c.transacaoID, fid); err != nil {
			return 0, err
		}
	}
	return len(categorizacoes), nil
}

// AplicarRegrasEmTodasAsFamilias roda a aplicacao para cada familia com regra.
func AplicarRegrasEmTodasAsFamilias(db *sql.DB) error {
	linhas, err := db.Query(`SELECT DISTINCT family_id FROM regras_de_categorizacao`)
	if err != nil {
		return err
	}
	var familias []int
	for linhas.Next() {
		var fid int
		if err := linhas.Scan(&fid); err != nil {
			linhas.Close()
			return err
		}
		familias = append(familias, fid)
	}
	linhas.Close()
	for _, fid := range familias {
		if _, err := AplicarRegrasDeCategorizacao(db, fid); err != nil {
			return err
		}
	}
	return nil
}

// CategoriaPelaPrimeiraRegra devolve a categoria da primeira regra (na ordem
// de prioridade) que casa com a transacao, ou "" se nenhuma casar.
func CategoriaPelaPrimeiraRegra(regras []models.RegraDeCategorizacao, descricao, categoriaDoBanco string) string {
	descricaoNormalizada := semAcentoEMinuscula(descricao)
	categoriaDoBancoNormalizada := semAcentoEMinuscula(categoriaDoBanco)
	for _, regra := range regras {
		texto := semAcentoEMinuscula(regra.Texto)
		switch regra.Campo {
		case "descricao":
			if strings.Contains(descricaoNormalizada, texto) {
				return regra.Categoria
			}
		case "categoria_do_banco":
			if categoriaDoBancoNormalizada == texto {
				return regra.Categoria
			}
		}
	}
	return ""
}

// ordenarPorPrioridade: regras de descricao antes das de categoria do banco
// (sao mais especificas); dentro do mesmo campo, texto mais longo primeiro
// ("UBER EATS" antes de "UBER"); empate, a regra mais antiga.
func ordenarPorPrioridade(regras []models.RegraDeCategorizacao) {
	sort.SliceStable(regras, func(i, j int) bool {
		if regras[i].Campo != regras[j].Campo {
			return regras[i].Campo == "descricao"
		}
		if len(regras[i].Texto) != len(regras[j].Texto) {
			return len(regras[i].Texto) > len(regras[j].Texto)
		}
		return regras[i].ID < regras[j].ID
	})
}

var removedorDeAcentos = transform.Chain(norm.NFD, runes.Remove(runes.In(unicode.Mn)), norm.NFC)

func semAcentoEMinuscula(texto string) string {
	semAcento, _, err := transform.String(removedorDeAcentos, texto)
	if err != nil {
		semAcento = texto
	}
	return strings.ToLower(strings.TrimSpace(semAcento))
}
