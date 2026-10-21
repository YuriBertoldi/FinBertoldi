package handler

import (
	"os"
	"testing"
)

// Os templates so sao compilados quando o app sobe (template.Must): funcao
// inexistente ou tag mal fechada passa no go build e derruba o app no
// deploy. Este teste compila todos no CI.
func TestTemplates_TodosCompilam(t *testing.T) {
	pastaDoTeste, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	// InitTemplates le "templates/..." relativo a raiz do projeto.
	if err := os.Chdir("../.."); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { os.Chdir(pastaDoTeste) })

	defer func() {
		if erro := recover(); erro != nil {
			t.Fatalf("um template nao compila: %v", erro)
		}
	}()
	InitTemplates()
}
