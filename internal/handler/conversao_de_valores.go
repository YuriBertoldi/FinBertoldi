package handler

import (
	"fmt"
	"strconv"
	"strings"
	"time"
)

func formatBRL(v float64) string {
	neg := v < 0
	if neg {
		v = -v
	}
	cents := int64(v*100+0.5) % 100
	whole := int64(v)
	s := strconv.FormatInt(whole, 10)
	if len(s) > 3 {
		var b strings.Builder
		for i, c := range s {
			if i > 0 && (len(s)-i)%3 == 0 {
				b.WriteRune('.')
			}
			b.WriteRune(c)
		}
		s = b.String()
	}
	result := fmt.Sprintf("R$ %s,%02d", s, cents)
	if neg {
		result = "-" + result
	}
	return result
}

func parseInt(s string) int {
	n, _ := strconv.Atoi(strings.TrimSpace(s))
	return n
}

func parseFloat(s string) float64 {
	s = strings.TrimSpace(s)
	hasDot := strings.Contains(s, ".")
	hasComma := strings.Contains(s, ",")
	if hasDot && hasComma {
		// Formato BR: 1.234,56 — ponto é milhar, vírgula é decimal
		s = strings.ReplaceAll(s, ".", "")
		s = strings.ReplaceAll(s, ",", ".")
	} else if hasComma {
		// Só vírgula: 223,05 → 223.05
		s = strings.ReplaceAll(s, ",", ".")
	}
	// Só ponto ou sem separador: já é float padrão (223.05)
	f, _ := strconv.ParseFloat(s, 64)
	return f
}

func parseDate(s string) time.Time {
	t, err := time.Parse("2006-01-02", strings.TrimSpace(s))
	if err != nil {
		return time.Now()
	}
	return t
}
