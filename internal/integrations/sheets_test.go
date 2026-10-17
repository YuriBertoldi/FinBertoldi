package integrations

import (
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"encoding/pem"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"fincontrol/internal/models"
)

func gerarServiceAccountJSON(t *testing.T, tokenURI string) (string, *rsa.PrivateKey) {
	t.Helper()
	chave, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}
	der, _ := x509.MarshalPKCS8PrivateKey(chave)
	pemKey := string(pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: der}))
	js, _ := json.Marshal(map[string]string{
		"type": "service_account", "client_email": "fin@proj.iam.gserviceaccount.com",
		"private_key": pemKey, "token_uri": tokenURI,
	})
	return string(js), chave
}

func TestParseServiceAccount(t *testing.T) {
	js, _ := gerarServiceAccountJSON(t, "")
	sa, err := ParseServiceAccount(js)
	if err != nil {
		t.Fatalf("JSON válido recusado: %v", err)
	}
	if sa.TokenURI != "https://oauth2.googleapis.com/token" {
		t.Errorf("token_uri padrão = %q", sa.TokenURI)
	}
	invalidos := map[string]string{
		"não é JSON":    "abc",
		"sem e-mail":    `{"private_key":"x"}`,
		"chave sem PEM": `{"client_email":"a@b","private_key":"nao-e-pem"}`,
		"PEM inválido":  `{"client_email":"a@b","private_key":"-----BEGIN PRIVATE KEY-----\nAAAA\n-----END PRIVATE KEY-----\n"}`,
	}
	for nome, v := range invalidos {
		if _, err := ParseServiceAccount(v); err == nil {
			t.Errorf("%s: deveria ser recusado", nome)
		}
	}
}

func TestAssinarJWT(t *testing.T) {
	js, chave := gerarServiceAccountJSON(t, "https://oauth2.googleapis.com/token")
	sa, _ := ParseServiceAccount(js)
	jwt, err := sa.assinarJWT(time.Unix(1_700_000_000, 0))
	if err != nil {
		t.Fatal(err)
	}
	partes := strings.Split(jwt, ".")
	if len(partes) != 3 {
		t.Fatalf("JWT com %d partes", len(partes))
	}
	assinatura, _ := base64.RawURLEncoding.DecodeString(partes[2])
	hash := sha256.Sum256([]byte(partes[0] + "." + partes[1]))
	if err := rsa.VerifyPKCS1v15(&chave.PublicKey, crypto.SHA256, hash[:], assinatura); err != nil {
		t.Fatalf("assinatura inválida: %v", err)
	}
	var claims map[string]any
	b, _ := base64.RawURLEncoding.DecodeString(partes[1])
	json.Unmarshal(b, &claims)
	if claims["iss"] != "fin@proj.iam.gserviceaccount.com" || claims["scope"] != sheetsScope ||
		claims["aud"] != "https://oauth2.googleapis.com/token" || claims["exp"].(float64)-claims["iat"].(float64) != 3600 {
		t.Errorf("claims inesperadas: %v", claims)
	}
}

func TestTokenAcesso(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		r.ParseForm()
		if r.Form.Get("grant_type") != "urn:ietf:params:oauth:grant-type:jwt-bearer" || strings.Count(r.Form.Get("assertion"), ".") != 2 {
			w.WriteHeader(400)
			io.WriteString(w, `{"error":"invalid_grant","error_description":"bad"}`)
			return
		}
		io.WriteString(w, `{"access_token":"tok-123","expires_in":3600}`)
	}))
	defer srv.Close()
	js, _ := gerarServiceAccountJSON(t, srv.URL)
	sa, _ := ParseServiceAccount(js)
	if tok, err := sa.TokenAcesso(); err != nil || tok != "tok-123" {
		t.Fatalf("TokenAcesso = %q, %v", tok, err)
	}

	recusa := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(400)
		io.WriteString(w, `{"error":"invalid_grant","error_description":"Invalid JWT Signature."}`)
	}))
	defer recusa.Close()
	js, _ = gerarServiceAccountJSON(t, recusa.URL)
	sa, _ = ParseServiceAccount(js)
	if _, err := sa.TokenAcesso(); err == nil || !strings.Contains(err.Error(), "Invalid JWT Signature") {
		t.Errorf("erro do Google deveria ser repassado, got %v", err)
	}
}

func TestExtrairIDPlanilha(t *testing.T) {
	cases := map[string]string{
		"1AbC_dEf-123":     "1AbC_dEf-123",
		"  1AbC_dEf-123  ": "1AbC_dEf-123",
		"https://docs.google.com/spreadsheets/d/1AbC_dEf-123/edit#gid=0":       "1AbC_dEf-123",
		"https://docs.google.com/spreadsheets/d/1AbC_dEf-123/edit?usp=sharing": "1AbC_dEf-123",
		"https://docs.google.com/spreadsheets/d/1AbC_dEf-123":                  "1AbC_dEf-123",
	}
	for in, want := range cases {
		if got := ExtrairIDPlanilha(in); got != want {
			t.Errorf("ExtrairIDPlanilha(%q) = %q; want %q", in, got, want)
		}
	}
}

func TestMontarAbas(t *testing.T) {
	d := DadosExport{
		Mes:         time.Date(2026, 10, 1, 0, 0, 0, 0, time.UTC),
		Agora:       time.Date(2026, 10, 17, 9, 30, 0, 0, time.UTC),
		ReceitasMes: 13000,
		Despesas:    []models.DespesaMes{{Nome: "Energia", Categoria: "Básicas", Valor: 500, Pago: true}},
		Parcelas: []models.Parcelamento{
			{Descricao: "Carro", Cartao: "Boleto", ValorParcela: 2514, ParcelaAtual: 3, TotalParcelas: 36},
			{Descricao: "Dufry", Cartao: "Nubank", ValorParcela: 74.47, ParcelaAtual: 1, TotalParcelas: 1},
		},
		Receitas: []models.Receita{{Descricao: "Salário", Valor: 13000, Data: time.Date(2026, 10, 5, 0, 0, 0, 0, time.UTC), Recorrente: true}},
	}
	abas := MontarAbas(d)
	var nomes []string
	por := map[string]Aba{}
	for _, a := range abas {
		nomes = append(nomes, a.Nome)
		por[a.Nome] = a
	}
	if strings.Join(nomes, ",") != "Resumo,Contas do mês,Receitas,Parcelamentos,Investimentos,Transações" {
		t.Fatalf("abas = %v", nomes)
	}
	contas := por["Contas do mês"].Linhas
	if len(contas) != 5 { // cabeçalho + 3 itens + total
		t.Fatalf("contas com %d linhas: %v", len(contas), contas)
	}
	if contas[2][3] != "3/36" || contas[3][0] != "À vista" || contas[3][3] != "" {
		t.Errorf("parcela/à vista mal classificados: %v / %v", contas[2], contas[3])
	}
	if total := contas[4][4].(float64); total != 3088.47 {
		t.Errorf("total = %v; want 3088.47", total)
	}
	resumo := por["Resumo"].Linhas
	if resumo[5][1].(float64) != 2588.47 || resumo[6][1].(float64) != 13000-3088.47 {
		t.Errorf("resumo a pagar/sobra errado: %v", resumo)
	}
	if r := por["Receitas"].Linhas[1]; r[0] != "05/10/2026" || r[3] != "Sim" {
		t.Errorf("receita = %v", r)
	}
	// abas sem dados ficam só com o cabeçalho (a limpeza remove dados antigos)
	if len(por["Investimentos"].Linhas) != 1 || len(por["Transações"].Linhas) != 1 {
		t.Error("abas vazias devem ter só o cabeçalho")
	}
}

func TestEnviarAbas(t *testing.T) {
	var chamadas []string
	var gravado map[string]any
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer tok" {
			w.WriteHeader(401)
			return
		}
		corpo, _ := io.ReadAll(r.Body)
		chamadas = append(chamadas, r.Method+" "+r.URL.Path+" "+string(corpo))
		switch {
		case r.Method == "GET":
			io.WriteString(w, `{"sheets":[{"properties":{"title":"Resumo"}},{"properties":{"title":"Página1"}}]}`)
		case strings.HasSuffix(r.URL.Path, "/values:batchUpdate"):
			json.Unmarshal(corpo, &gravado)
			io.WriteString(w, `{}`)
		default:
			io.WriteString(w, `{}`)
		}
	}))
	defer srv.Close()
	antes := sheetsAPIBase
	sheetsAPIBase = srv.URL
	defer func() { sheetsAPIBase = antes }()

	abas := []Aba{{"Resumo", [][]any{{"Item", "Valor"}}}, {"Contas do mês", [][]any{{"Tipo"}, {"Fixa"}}}}
	if err := EnviarAbas("tok", "planilha1", abas); err != nil {
		t.Fatalf("EnviarAbas: %v", err)
	}
	if len(chamadas) != 4 {
		t.Fatalf("esperava 4 chamadas (ler, criar aba, limpar, gravar), got %d: %v", len(chamadas), chamadas)
	}
	if !strings.Contains(chamadas[1], ":batchUpdate") || !strings.Contains(chamadas[1], `"title":"Contas do mês"`) || strings.Contains(chamadas[1], `"title":"Resumo"`) {
		t.Errorf("deveria criar só a aba que falta: %s", chamadas[1])
	}
	if !strings.Contains(chamadas[2], "values:batchClear") || !strings.Contains(chamadas[2], "Contas do mês") {
		t.Errorf("deveria limpar as abas antes de gravar: %s", chamadas[2])
	}
	if gravado["valueInputOption"] != "USER_ENTERED" || len(gravado["data"].([]any)) != 2 {
		t.Errorf("gravação inesperada: %v", gravado)
	}
}

func TestEnviarAbasErros(t *testing.T) {
	for status, trecho := range map[int]string{403: "compartilhe", 404: "não encontrada"} {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(status) }))
		antes := sheetsAPIBase
		sheetsAPIBase = srv.URL
		err := EnviarAbas("tok", "x", []Aba{{"Resumo", nil}})
		sheetsAPIBase = antes
		srv.Close()
		if err == nil || !strings.Contains(err.Error(), trecho) {
			t.Errorf("HTTP %d: erro = %v; deveria explicar (%q)", status, err, trecho)
		}
	}
}
