package store

import (
	"testing"

	"fincontrol/internal/models"
)

// A ordem e o casamento decidem em que categoria cada gasto cai, e com isso o
// orcamento. Cada caso e uma regra de escolha que, quebrada, joga gasto na
// categoria errada.
func TestCategoriaPelaPrimeiraRegra(t *testing.T) {
	regras := []models.RegraDeCategorizacao{
		{ID: 1, Campo: "categoria_do_banco", Texto: "Groceries", Categoria: "Alimentação"},
		{ID: 2, Campo: "categoria_do_banco", Texto: "Eating out", Categoria: "Lazer"},
		{ID: 3, Campo: "descricao", Texto: "UBER", Categoria: "Transporte"},
		{ID: 4, Campo: "descricao", Texto: "UBER EATS", Categoria: "Lazer"},
		{ID: 5, Campo: "descricao", Texto: "farmácia", Categoria: "Saúde"},
	}
	ordenarPorPrioridade(regras)

	casos := []struct {
		nome, descricao, categoriaDoBanco, esperada string
	}{
		{"categoria do banco sem diferenca de maiusculas", "COMPRA 123", "GROCERIES", "Alimentação"},
		{"categoria do banco precisa ser igual, nao basta conter", "COMPRA", "Groceries and more", ""},
		{"descricao contida, sem diferenca de acentos", "DROGARIA FARMACIA SAO JOAO", "Health", "Saúde"},
		{"descricao vale antes da categoria do banco", "FARMACIA PAGUE MENOS", "Groceries", "Saúde"},
		{"texto mais longo vale antes do mais curto", "UBER EATS *PEDIDO", "Eating out", "Lazer"},
		{"texto curto continua valendo sozinho", "UBER *TRIP", "Transportation", "Transporte"},
		{"nenhuma regra casa", "PIX RECEBIDO", "Transfers", ""},
	}
	for _, caso := range casos {
		t.Run(caso.nome, func(t *testing.T) {
			if obtida := CategoriaPelaPrimeiraRegra(regras, caso.descricao, caso.categoriaDoBanco); obtida != caso.esperada {
				t.Errorf("categoria = %q; esperado %q", obtida, caso.esperada)
			}
		})
	}
}
