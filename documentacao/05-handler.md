# Modulo: handler

**Pacote:** `internal/handler`
**Arquivos:** `handler.go`, `bankimport.go`, `reports.go`, `importexport.go`

## Responsabilidade

Todos os handlers HTTP do sistema. Recebe requests, processa dados, chama store, renderiza templates.

## Arquivos

### handler.go (principal)

Contem:
- `InitTemplates()` — carrega e compila todos os templates HTML
- `render(w, template, data)` — renderiza template com dados
- Helpers de parsing: `parseFloat`, `parseInt`, `parseDate`, `parseBool`, `boolStr`, `str`
- `mesFromRequest(r)` — extrai mes da query string com fallback para mes atual
- `mesDisplay(mes)` — converte "2026-01" para "Janeiro/2026"
- `formatBRL(v)` — formata float como "R$ 1.234,56"
- `calcResumo(...)` — calcula totais do dashboard (despesas, pago, pendente, sobra, caixa)
- Handlers de todas as telas (Dashboard, Despesas, Receitas, etc)
- Handlers de integracao Pluggy (admin only)
- Handlers de recuperacao de senha (forgot/reset)
- Handlers de integracoes (listar, salvar, testar)

### bankimport.go

Parsers de importacao bancaria e handlers de transacoes:
- `parseCSVFile(r)` — parser de CSV com auto-deteccao de separador (virgula/ponto-e-virgula) e formato BR
- `parseOFXFile(r)` — parser de OFX via biblioteca ofxgo
- `parseDateFlex(s)` — parse de data em multiplos formatos (DD/MM/YYYY, YYYY-MM-DD, etc)
- `parseValorBR(s)` — parse de valor monetario BR ("1.234,56") e internacional ("1,234.56")
- `normalizeHeader(s)` — normaliza cabecalho CSV (lowercase, remove BOM, strip non-printable)
- `HandleTransacoes` — lista transacoes com filtros
- `HandleImportCSV` — upload e parse de CSV
- `HandleImportOFX` — upload e parse de OFX
- `HandleTransacaoConverter` — converte transacao em despesa ou receita
- `HandleTransacaoIgnorar` — marca transacao como ignorada
- `HandleTransacaoCategorizar` — atribui categoria a transacao
- `HandleTransacaoVincular` — vincula transacao a despesa/receita existente
- `HandleTransacaoDesvincular` — remove vinculo de transacao
- `HandleAutoMatch` — auto-match de transacoes pendentes por valor/data
- `HandleTransacaoMatches` — endpoint HTMX com sugestoes de match

### reports.go

Handlers de relatorios PDF (HTML otimizado para impressao):
- `HandleRelatorioDashboard`
- `HandleRelatorioDespesas`
- `HandleRelatorioReceitas`
- `HandleRelatorioInvestimentos`
- `HandleRelatorioEmprestimos`

### importexport.go

Import/Export XLSX via excelize:
- `HandleImportPage` — pagina de import/export
- `HandleExport` — exporta dados atuais em XLSX (5 abas)
- `HandleExportTemplate` — modelo em branco para preenchimento
- `HandleImport` — importa XLSX com dados

## Padrao dos handlers

Todos seguem o padrao:

```go
func HandleX(db *sql.DB) http.HandlerFunc {
    return func(w http.ResponseWriter, r *http.Request) {
        u := auth.CurrentUser(r)
        fid := u.FamilyID
        // ... logica ...
        render(w, "template-name", PageData{...})
    }
}
```

## Template helpers (FuncMap)

| Helper | Descricao |
|--------|-----------|
| `brl` | Formata float como HTML span com classe: `<span class="money">R$ 1.234,56</span>` |
| `date` | Formata `time.Time` como "02/01/2006" |
| `dateInput` | Formata para input HTML: "2006-01-02" |
| `neg` | Retorna true se valor < 0 |
| `abs` | Valor absoluto |
| `sub` | Subtracao: `sub a b` = a - b |
| `pct` | Percentual: `pct parte total` = parte/total*100 |
| `progress` | Barra de progresso CSS |
| `seq` | Gera sequencia de inteiros |
| `json` | Serializa para JSON (usado em Chart.js) |
| `contains` | Verifica se slice contem elemento |
| `deref` | Dereferencia ponteiro *float64 (nil → 0) |
| `mesDisplay` | "2026-01" → "Janeiro/2026" |
| `mesNav` | Navegacao de mes (anterior/proximo) |

## Handlers Pluggy (admin only)

| Handler | Rota | Descricao |
|---------|------|-----------|
| `HandleSavePluggyConfig` | POST /cadastros/pluggy/config | Salva credenciais Pluggy |
| `HandlePluggyConnectToken` | POST /cadastros/pluggy/connect-token | Proxy para token do Connect Widget |
| `HandlePluggyRegisterItem` | POST /cadastros/pluggy/items | Registra item conectado |
| `HandlePluggyDisconnect` | POST /cadastros/pluggy/{item_id}/disconnect | Remove conta conectada |
| `HandlePluggySync` | POST /cadastros/pluggy/{item_id}/sync | Forca sync de transacoes |

Funcoes auxiliares:
- `pluggyHealthCheck(url)` — verifica se o microservico esta online
- `pluggyProxy(db, method, path, body)` — proxy HTTP para o pluggy-service

## Testes

`handler_test.go` — testa:
- `formatBRL` — formatacao monetaria BR
- `parseFloat` — parse de floats BR e internacionais
- `parseInt` — parse de inteiros
- `parseDate` — parse de datas
- `mesDisplay` — display de mes
- `mesFromRequest` — extracao de mes da request
- `boolStr` / `parseBool` — conversao de booleanos
- `str` — acesso seguro a slice de strings
- `calcResumo` — calculo dos cards do dashboard
- `parseDateFlex` — parse de datas em multiplos formatos
- `parseValorBR` — parse de valores monetarios BR
- `normalizeHeader` — normalizacao de cabecalhos CSV
- `parseCSVFile` — parse completo de CSV (comma, semicolon, com categoria, vazio, dedup)

## Handlers de Recuperacao de Senha

| Handler | Rota | Descricao |
|---------|------|-----------|
| `HandleForgotPassword` | GET/POST /forgot-password | Formulario e envio de email com token |
| `HandleResetPassword` | GET/POST /reset-senha | Validacao de token e atualizacao de senha |

## Handlers de Integracoes

| Handler | Rota | Descricao |
|---------|------|-----------|
| `HandleIntegracoes` | GET /integracoes | Pagina de configuracao de integracoes |
| `HandleIntegracaoSalvar` | POST /integracoes/{nome} | Salvar config (KV) e toggle ativa |
| `HandleIntegracaoTestar` | POST /integracoes/{nome}/testar | Testar integracao (retorna JSON) |
