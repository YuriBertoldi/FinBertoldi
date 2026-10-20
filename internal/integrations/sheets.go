package integrations

import (
	"bytes"
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"database/sql"
	"encoding/base64"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"strings"
	"time"

	"fincontrol/internal/models"
	"fincontrol/internal/store"
)

// Endereços da API — variáveis para os testes apontarem para um servidor local.
var (
	sheetsAPIBase = "https://sheets.googleapis.com/v4/spreadsheets"
	sheetsHTTP    = &http.Client{Timeout: 30 * time.Second}
)

const sheetsScope = "https://www.googleapis.com/auth/spreadsheets"

// ServiceAccount são os campos usados do JSON de chave baixado do Google Cloud.
type ServiceAccount struct {
	ClientEmail string `json:"client_email"`
	PrivateKey  string `json:"private_key"`
	TokenURI    string `json:"token_uri"`
	chave       *rsa.PrivateKey
}

// ParseServiceAccount valida o JSON da service account e carrega a chave privada.
func ParseServiceAccount(js string) (*ServiceAccount, error) {
	var sa ServiceAccount
	if err := json.Unmarshal([]byte(js), &sa); err != nil {
		return nil, fmt.Errorf("JSON da service account inválido: cole o arquivo .json inteiro baixado do Google Cloud")
	}
	if sa.ClientEmail == "" || sa.PrivateKey == "" {
		return nil, fmt.Errorf("JSON da service account sem client_email ou private_key")
	}
	if sa.TokenURI == "" {
		sa.TokenURI = "https://oauth2.googleapis.com/token"
	}
	bloco, _ := pem.Decode([]byte(sa.PrivateKey))
	if bloco == nil {
		return nil, fmt.Errorf("private_key da service account não está no formato PEM")
	}
	var chave any
	var err error
	if chave, err = x509.ParsePKCS8PrivateKey(bloco.Bytes); err != nil {
		if chave, err = x509.ParsePKCS1PrivateKey(bloco.Bytes); err != nil {
			return nil, fmt.Errorf("não foi possível ler a private_key da service account")
		}
	}
	rsaChave, ok := chave.(*rsa.PrivateKey)
	if !ok {
		return nil, fmt.Errorf("a private_key da service account precisa ser RSA")
	}
	sa.chave = rsaChave
	return &sa, nil
}

// assinarJWT monta o JWT (RS256) que o Google troca por um token de acesso.
func (sa *ServiceAccount) assinarJWT(agora time.Time) (string, error) {
	b64 := base64.RawURLEncoding.EncodeToString
	cab, _ := json.Marshal(map[string]string{"alg": "RS256", "typ": "JWT"})
	claims, _ := json.Marshal(map[string]any{
		"iss":   sa.ClientEmail,
		"scope": sheetsScope,
		"aud":   sa.TokenURI,
		"iat":   agora.Unix(),
		"exp":   agora.Add(time.Hour).Unix(),
	})
	conteudo := b64(cab) + "." + b64(claims)
	hash := sha256.Sum256([]byte(conteudo))
	assinatura, err := rsa.SignPKCS1v15(rand.Reader, sa.chave, crypto.SHA256, hash[:])
	if err != nil {
		return "", err
	}
	return conteudo + "." + b64(assinatura), nil
}

// TokenAcesso troca o JWT assinado por um token de acesso OAuth.
func (sa *ServiceAccount) TokenAcesso() (string, error) {
	jwt, err := sa.assinarJWT(time.Now())
	if err != nil {
		return "", fmt.Errorf("assinar JWT: %w", err)
	}
	resp, err := sheetsHTTP.PostForm(sa.TokenURI, url.Values{
		"grant_type": {"urn:ietf:params:oauth:grant-type:jwt-bearer"},
		"assertion":  {jwt},
	})
	if err != nil {
		return "", fmt.Errorf("falha ao contatar o Google: %w", err)
	}
	defer resp.Body.Close()
	var r struct {
		AccessToken string `json:"access_token"`
		Erro        string `json:"error"`
		Descricao   string `json:"error_description"`
	}
	json.NewDecoder(resp.Body).Decode(&r)
	if resp.StatusCode != http.StatusOK || r.AccessToken == "" {
		return "", fmt.Errorf("service account recusada pelo Google (%s %s) — confira se a chave não foi revogada", r.Erro, r.Descricao)
	}
	return r.AccessToken, nil
}

// ExtrairIDPlanilha aceita o ID ou o link completo da planilha.
func ExtrairIDPlanilha(s string) string {
	s = strings.TrimSpace(s)
	if i := strings.Index(s, "/d/"); i >= 0 {
		s = s[i+3:]
		if j := strings.IndexAny(s, "/?#"); j >= 0 {
			s = s[:j]
		}
	}
	return s
}

// Aba é uma aba da planilha com suas linhas (a primeira é o cabeçalho).
type Aba struct {
	Nome   string
	Linhas [][]any
}

// DadosExport reúne o que vai para a planilha.
type DadosExport struct {
	Mes           time.Time
	Despesas      []models.DespesaMes
	Parcelas      []models.Parcelamento // vigentes no mês (inclui compras à vista 1/1)
	Receitas      []models.Receita
	ReceitasMes   float64
	Parcelamentos []models.Parcelamento // ativos, com restante
	Investimentos []models.Investimento
	Transacoes    []models.TransacaoBanco
	Agora         time.Time
}

func simNao(b bool) string {
	if b {
		return "Sim"
	}
	return "Não"
}

func situacao(pago bool) string {
	if pago {
		return "Pago"
	}
	return "Pendente"
}

func data(t time.Time) string { return t.Format("02/01/2006") }

// MontarAbas transforma os dados nas abas da planilha.
func MontarAbas(d DadosExport) []Aba {
	contas := [][]any{{"Tipo", "Descrição", "Categoria / Cartão", "Parcela", "Valor", "Situação"}}
	var totalContas, totalPago float64
	for _, x := range d.Despesas {
		contas = append(contas, []any{"Fixa", x.Nome, x.Categoria, "", x.Valor, situacao(x.Pago)})
		totalContas += x.Valor
		if x.Pago {
			totalPago += x.Valor
		}
	}
	for _, p := range d.Parcelas {
		tipo, parcela := "Parcelamento", fmt.Sprintf("%d/%d", p.ParcelaAtual, p.TotalParcelas)
		if p.TotalParcelas <= 1 {
			tipo, parcela = "À vista", ""
		}
		contas = append(contas, []any{tipo, p.Descricao, p.Cartao, parcela, p.ValorParcela, situacao(p.Pago)})
		totalContas += p.ValorParcela
		if p.Pago {
			totalPago += p.ValorParcela
		}
	}
	contas = append(contas, []any{"", "TOTAL", "", "", totalContas, ""})

	resumo := [][]any{
		{"Item", "Valor"},
		{"Mês", d.Mes.Format("01/2006")},
		{"Receitas do mês", d.ReceitasMes},
		{"Contas do mês", totalContas},
		{"Já pago", totalPago},
		{"A pagar", totalContas - totalPago},
		{"Sobra", d.ReceitasMes - totalContas},
		{"Atualizado em", d.Agora.Format("02/01/2006 15:04")},
	}

	receitas := [][]any{{"Data", "Descrição", "Tipo", "Recorrente", "Valor"}}
	for _, r := range d.Receitas {
		receitas = append(receitas, []any{data(r.Data), r.Descricao, r.Tipo, simNao(r.Recorrente), r.Valor})
	}

	parcs := [][]any{{"Descrição", "Cartão", "Parcela atual", "Total de parcelas", "Valor da parcela", "Início", "Restantes", "Valor restante"}}
	for _, p := range d.Parcelamentos {
		parcs = append(parcs, []any{p.Descricao, p.Cartao, p.ParcelaAtual, p.TotalParcelas, p.ValorParcela, data(p.DataInicio), p.Restantes, p.ValorRestante})
	}

	invs := [][]any{{"Data", "Instituição", "Tipo", "Valor", "Notas"}}
	for _, i := range d.Investimentos {
		invs = append(invs, []any{data(i.Data), i.Instituicao, i.Tipo, i.Valor, i.Notas})
	}

	txns := [][]any{{"Data", "Descrição", "Valor", "Tipo", "Categoria", "Situação", "Banco"}}
	for _, t := range d.Transacoes {
		txns = append(txns, []any{data(t.Data), t.Descricao, t.Valor, t.Tipo, t.Categoria, t.Status, t.Banco})
	}

	return []Aba{
		{"Resumo", resumo},
		{"Contas do mês", contas},
		{"Receitas", receitas},
		{"Parcelamentos", parcs},
		{"Investimentos", invs},
		{"Transações", txns},
	}
}

func carregarDados(db *sql.DB, fid int, agora time.Time) DadosExport {
	mes := time.Date(agora.Year(), agora.Month(), 1, 0, 0, 0, 0, time.UTC)
	mesStr := mes.Format("2006-01")
	d := DadosExport{Mes: mes, Agora: agora}
	store.EnsureDespesasMes(db, fid, mes)
	basicas, cartao, vr, _ := store.GetDespesasMes(db, fid, mes, mesStr)
	d.Despesas = append(append(basicas, cartao...), vr...)
	d.Parcelas, _ = store.GetParcelamentosVigentesNoMes(db, fid, mes)
	d.ReceitasMes, _ = store.GetReceitasMes(db, fid, mes)
	d.Receitas, _ = store.GetReceitas(db, fid)
	d.Parcelamentos, _ = store.GetParcelamentos(db, fid, true)
	d.Investimentos, _ = store.GetInvestimentos(db, fid)
	d.Transacoes, _ = store.GetTransacoesBanco(db, fid, mesStr, "", "", "")
	return d
}

// ExportToSheet atualiza a planilha configurada da família com os dados atuais.
func ExportToSheet(db *sql.DB, fid int) error {
	id := ExtrairIDPlanilha(store.GetIntegracaoKV(db, fid, "sheets", "spreadsheet_id"))
	saJSON := store.GetIntegracaoKV(db, fid, "sheets", "service_account_json")
	if id == "" || saJSON == "" {
		return fmt.Errorf("informe o ID da planilha e o JSON da service account")
	}
	sa, err := ParseServiceAccount(saJSON)
	if err != nil {
		return err
	}
	token, err := sa.TokenAcesso()
	if err != nil {
		return err
	}
	agora := time.Now()
	if err := EnviarAbas(token, id, MontarAbas(carregarDados(db, fid, agora))); err != nil {
		return err
	}
	store.SetIntegracaoKV(db, fid, "sheets", "ultima_exportacao", agora.Format("02/01/2006 15:04"))
	log.Printf("[sheets] família %d exportada", fid)
	return nil
}

// chamarSheets faz uma chamada autenticada à API e traduz os erros mais comuns.
func chamarSheets(token, metodo, endereco string, corpo any, saida any) error {
	var leitor io.Reader
	if corpo != nil {
		b, _ := json.Marshal(corpo)
		leitor = bytes.NewReader(b)
	}
	req, err := http.NewRequest(metodo, endereco, leitor)
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")
	resp, err := sheetsHTTP.Do(req)
	if err != nil {
		return fmt.Errorf("falha ao contatar o Google Sheets: %w", err)
	}
	defer resp.Body.Close()
	switch resp.StatusCode {
	case http.StatusOK:
		if saida != nil {
			return json.NewDecoder(resp.Body).Decode(saida)
		}
		return nil
	case http.StatusForbidden:
		return fmt.Errorf("sem permissão na planilha — compartilhe-a com o e-mail da service account como Editor e ative a Google Sheets API no projeto")
	case http.StatusNotFound:
		return fmt.Errorf("planilha não encontrada — confira o ID/link da planilha")
	}
	var e struct {
		Error struct {
			Message string `json:"message"`
		} `json:"error"`
	}
	json.NewDecoder(resp.Body).Decode(&e)
	return fmt.Errorf("resposta %d do Google Sheets: %s", resp.StatusCode, e.Error.Message)
}

// EnviarAbas cria as abas que faltam, limpa e grava cada uma.
func EnviarAbas(token, planilhaID string, abas []Aba) error {
	base := sheetsAPIBase + "/" + url.PathEscape(planilhaID)

	var meta struct {
		Sheets []struct {
			Properties struct {
				Title string `json:"title"`
			} `json:"properties"`
		} `json:"sheets"`
	}
	if err := chamarSheets(token, "GET", base+"?fields=sheets.properties.title", nil, &meta); err != nil {
		return err
	}
	existe := map[string]bool{}
	for _, s := range meta.Sheets {
		existe[s.Properties.Title] = true
	}
	var novas []map[string]any
	for _, a := range abas {
		if !existe[a.Nome] {
			novas = append(novas, map[string]any{"addSheet": map[string]any{"properties": map[string]any{"title": a.Nome}}})
		}
	}
	if len(novas) > 0 {
		if err := chamarSheets(token, "POST", base+":batchUpdate", map[string]any{"requests": novas}, nil); err != nil {
			return err
		}
	}

	var intervalos []string
	for _, a := range abas {
		intervalos = append(intervalos, "'"+a.Nome+"'")
	}
	if err := chamarSheets(token, "POST", base+"/values:batchClear", map[string]any{"ranges": intervalos}, nil); err != nil {
		return err
	}
	var dados []map[string]any
	for _, a := range abas {
		dados = append(dados, map[string]any{"range": "'" + a.Nome + "'!A1", "values": a.Linhas})
	}
	return chamarSheets(token, "POST", base+"/values:batchUpdate",
		map[string]any{"valueInputOption": "USER_ENTERED", "data": dados}, nil)
}
