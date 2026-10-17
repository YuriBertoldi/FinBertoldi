// Package contas monta o resumo das contas do mês (formato WhatsApp) e o envia
// por e-mail e Telegram, sob demanda ou mensalmente.
package contas

import (
	"database/sql"
	"fmt"
	"sort"
	"strings"
	"time"
	"unicode/utf8"

	"fincontrol/internal/models"
	"fincontrol/internal/store"
)

// Largura das linhas dentro dos blocos monoespaçados — cabe na tela do celular.
const waLargura = 26

var nomesMes = []string{"", "Janeiro", "Fevereiro", "Março", "Abril", "Maio", "Junho",
	"Julho", "Agosto", "Setembro", "Outubro", "Novembro", "Dezembro"}

// MesDisplay devolve "Outubro/2026".
func MesDisplay(mes time.Time) string {
	return fmt.Sprintf("%s/%d", nomesMes[mes.Month()], mes.Year())
}

type waItem struct {
	nome  string
	valor float64
	pago  bool
}

// Texto monta as contas do mês da família no formato do WhatsApp.
func Texto(db *sql.DB, fid int, mes time.Time) string {
	mes = time.Date(mes.Year(), mes.Month(), 1, 0, 0, 0, 0, time.UTC)
	store.EnsureDespesasMes(db, fid, mes)
	basicas, cartao, vr, _ := store.GetDespesasMes(db, fid, mes, mes.Format("2006-01"))
	parcs, _ := store.GetParcelamentosVigentesNoMes(db, fid, mes)
	receitas, _ := store.GetReceitasMes(db, fid, mes)
	return BuildTexto(MesDisplay(mes), append(append(basicas, cartao...), vr...), parcs, receitas)
}

// BuildTexto gera o texto a partir dos dados já carregados.
func BuildTexto(mes string, despesas []models.DespesaMes, parcs []models.Parcelamento, receitas float64) string {
	var fixas, parcelados, avista []waItem
	for _, d := range despesas {
		fixas = append(fixas, waItem{d.Nome, d.Valor, d.Pago})
	}
	for _, p := range parcs {
		if p.TotalParcelas <= 1 {
			avista = append(avista, waItem{p.Descricao, p.ValorParcela, p.Pago})
			continue
		}
		frac := fmt.Sprintf(" %d/%d", p.ParcelaAtual, p.TotalParcelas)
		espaco := waLargura - utf8.RuneCountInString(waValor(p.ValorParcela)) - 1 - utf8.RuneCountInString(frac)
		parcelados = append(parcelados, waItem{waCorta(p.Descricao, espaco) + frac, p.ValorParcela, p.Pago})
	}

	var b strings.Builder
	fmt.Fprintf(&b, "*💰 Contas – %s*\n", mes)

	var total, pago float64
	secoes := []struct {
		titulo string
		itens  []waItem
		soma   float64
	}{
		{"🏠 FIXAS", fixas, 0},
		{"📦 PARCELAMENTOS", parcelados, 0},
		{"💳 COMPRAS À VISTA", avista, 0},
	}
	for i := range secoes {
		s := &secoes[i]
		if len(s.itens) == 0 {
			continue
		}
		sort.SliceStable(s.itens, func(a, c int) bool { return s.itens[a].valor > s.itens[c].valor })
		fmt.Fprintf(&b, "\n*%s*\n```\n", s.titulo)
		for _, it := range s.itens {
			b.WriteString(waLinha(it.nome, it.valor))
			s.soma += it.valor
			if it.pago {
				pago += it.valor
			}
		}
		b.WriteString(strings.Repeat("-", waLargura) + "\n")
		b.WriteString(waLinha("Subtotal", s.soma))
		b.WriteString("```\n")
		total += s.soma
	}

	b.WriteString("\n*📊 RESUMO*\n```\n")
	for _, s := range secoes {
		if len(s.itens) > 0 {
			b.WriteString(waLinha(strings.TrimSpace(s.titulo[strings.Index(s.titulo, " "):]), s.soma))
		}
	}
	b.WriteString(strings.Repeat("=", waLargura) + "\n")
	b.WriteString(waLinha("TOTAL", total))
	b.WriteString(waLinha("Já pago", pago))
	b.WriteString(waLinha("A pagar", total-pago))
	if receitas > 0 {
		b.WriteString(waLinha("Receitas", receitas))
		b.WriteString(waLinha("Sobra", receitas-total))
	}
	b.WriteString("```")
	return b.String()
}

// waLinha alinha o nome à esquerda e o valor à direita em waLargura colunas.
func waLinha(nome string, valor float64) string {
	v := waValor(valor)
	nome = waCorta(nome, waLargura-utf8.RuneCountInString(v)-1)
	pad := waLargura - utf8.RuneCountInString(nome) - utf8.RuneCountInString(v)
	if pad < 1 {
		pad = 1
	}
	return nome + strings.Repeat(" ", pad) + v + "\n"
}

// waValor formata em reais sem o "R$" para economizar colunas: 1.234,56
func waValor(v float64) string {
	neg := v < 0
	if neg {
		v = -v
	}
	cents := int64(v*100 + 0.5)
	s := fmt.Sprintf("%d", cents/100)
	var b strings.Builder
	for i, c := range s {
		if i > 0 && (len(s)-i)%3 == 0 {
			b.WriteRune('.')
		}
		b.WriteRune(c)
	}
	out := fmt.Sprintf("%s,%02d", b.String(), cents%100)
	if neg {
		out = "-" + out
	}
	return out
}

func waCorta(s string, max int) string {
	s = strings.TrimSpace(s)
	if max < 1 || utf8.RuneCountInString(s) <= max {
		return s
	}
	return strings.TrimSpace(string([]rune(s)[:max]))
}
