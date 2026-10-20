# Modulo: handler

**Pacote:** `internal/handler`
**Arquivos:** um por tela ou assunto (tabela abaixo)

## Responsabilidade

Todos os handlers HTTP do sistema. Recebe requests, processa dados, chama store, renderiza templates.

## Arquivos

### Mapa dos arquivos

| Arquivo | Conteudo |
|---|---|
| `templates.go` | `InitTemplates()`, `render`, `renderLogin`, `renderStandalone` e funcoes dos templates |
| `pagina_base.go` | `bp` (dados comuns da pagina), `mesFromRequest`, `mesDisplay` |
| `conversao_de_valores.go` | `formatBRL`, `parseInt`, `parseFloat`, `parseDate` |
| `autenticacao.go` | Login (com limite de tentativas), logout, login com Google, cadastro, recuperacao e redefinicao de senha |
| `minha_senha.go` | Troca da propria senha e `validarNovaSenha` (regra usada tambem por usuarios e minha familia) |
| `dashboard.go` | Dashboard, widgets, `calcResumo` e marcar pago pelo dashboard (HTMX) |
| `despesas.go` | Despesas fixas |
| `parcelamentos.go` | Parcelamentos e antecipacao de parcelas |
| `receitas.go` | Receitas |
| `investimentos.go` | Investimentos e reserva de emergencia |
| `pagamentos.go` | Pagamentos do mes |
| `emprestimos.go` | Emprestimos e pagamentos parciais |
| `planejamento.go` | Planejamento FIRE |
| `cadastros.go` | Categorias e cartoes |
| `usuarios.go` | Usuarios (admin geral) |
| `familias.go` | Familias (admin geral) |
| `minha_familia.go` | Membros e permissao por tela (`TelasDispo`) |
| `integracoes.go` | Integracoes gratuitas: listar, salvar, testar |
| `integracao_pluggy.go` | Administracao da integracao Pluggy (proxy para o pluggy-service) |
| `saude.go` | `/healthz` |
| `bankimport.go` | Importacao CSV/OFX e conciliacao de transacoes |
| `reports.go` | Relatorios PDF |
| `importexport.go` | Importar/Exportar XLSX |
| `whatsapp.go` | Contas do mes (WhatsApp/e-mail/Telegram) e notificacoes |

Regra: handler novo vai no arquivo da tela dele; funcao usada por varias telas fica no arquivo do assunto principal dela (ex.: `validarNovaSenha` em `minha_senha.go`). Nada de `utils.go`.

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

Ao salvar uma integracao: valida o token do Telegram (`ValidarTokenTelegram`) e o JSON da Service Account (`ParseServiceAccount`); normaliza o link da planilha (`ExtrairIDPlanilha`); segredos (`bot_token`, `service_account_json`) em branco mantem o valor salvo (`valorParaSalvar`) e nunca sao enviados de volta para a pagina. Com a integracao ativa, o Telegram e testado e a planilha e exportada na hora.

### whatsapp.go — contas do mes e notificacoes

| Handler | Rota | Descricao |
|---------|------|-----------|
| `HandlePagamentosWhatsApp` | GET /pagamentos/whatsapp | Texto das contas do mes (formato WhatsApp) |
| `HandlePagamentosEnviar` | POST /pagamentos/enviar | Envia ao usuario logado; `canal=telegram\|email` (sem canal: os dois). Responde HTML para HTMX |
| `HandleNotificacoes` | GET /notificacoes | Preferencias de envio mensal + @ do bot + mes que sera enviado |
| `HandleSaveNotificacoes` | POST /notificacoes | Salva e-mail/Telegram, dia (1–31), mes atual/seguinte e Chat ID |

Funcoes auxiliares testadas: `canaisEnvio`, `resultadoEnvioHTML` (escapa o erro), `valorParaSalvar`, `validarNovaSenha`, `montarLayoutWidgets`.

## Senhas

- `HandleMinhaSenha` — exige a senha atual; `validarNovaSenha` (minimo 8 caracteres, diferente da atual, recusa senhas conhecidas); usa `store.TrocarSenha` mantendo a sessao atual e encerrando as outras.
- `HandleResetSenha` (admin) e `HandleMeuTimeResetSenha` (admin de familia) — mesma validacao; encerram todas as sessoes do usuario.

## Dashboard: widgets

`HandleDashboard` carrega `store.GetDashboardLayout` e separa as chaves visiveis em `WidgetsEsquerda`/`WidgetsDireita`. `HandleSaveWidgets` aplica a ordem (`ordem`), os marcados (`widgets`) e a coluna (`coluna_<chave>`) via `montarLayoutWidgets`.

## Template helpers adicionais

- `assetV` — versao do `static/app.css` (CRC32 do arquivo). O `base.html` usa `/static/app.css?v={{assetV}}` para o navegador nao reaproveitar CSS antigo apos um deploy.
- `dict` — monta um mapa para passar mais de um valor a um template (`{{template "x" (dict "K" $k "P" $)}}`).
