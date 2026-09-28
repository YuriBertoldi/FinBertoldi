# Mapa de Rotas

**Arquivo:** `main.go`

## Publicas (sem autenticacao)

| Metodo | Rota | Handler | Descricao |
|--------|------|---------|-----------|
| GET | `/static/*` | FileServer | Arquivos estaticos (CSS, JS) |
| GET | `/login` | HandleLogin | Tela de login |
| POST | `/login` | HandleLogin | Processar login |
| POST | `/logout` | HandleLogout | Encerrar sessao |

## Protegidas (Protected — requer login)

| Metodo | Rota | Handler | Descricao |
|--------|------|---------|-----------|
| GET | `/` | HandleDashboard | Dashboard |
| POST | `/api/despesas/{id}/toggle-pago` | HandleToggleDespesaPago | Toggle pago no dashboard |
| POST | `/api/parcelamentos/{id}/toggle-pago` | HandleToggleParcelamentoPago | Toggle pago no dashboard |
| GET | `/dados` | HandleImportPage | Pagina import/export |
| GET | `/dados/modelo` | HandleExportTemplate | Download modelo XLSX |
| GET | `/dados/exportar` | HandleExport | Download dados XLSX |
| POST | `/dados/importar` | HandleImport | Upload XLSX |
| GET | `/relatorio/dashboard` | HandleRelatorioDashboard | Relatorio PDF |
| POST | `/minha-senha` | HandleMinhaSenha | Alterar propria senha |

## ScreenProtected (controle de acesso por tela)

### Despesas (`screen: "despesas"`)

| Metodo | Rota | Handler |
|--------|------|---------|
| GET | `/despesas` | HandleDespesas |
| POST | `/despesas` | HandleCreateDespesa |
| POST | `/despesas/{id}/update` | HandleUpdateDespesa |
| POST | `/despesas/{id}/delete` | HandleDeleteDespesa |
| POST | `/despesas/{id}/toggle-ativa` | HandleToggleDespesaAtiva |
| GET | `/parcelamentos` | HandleParcelamentos |
| POST | `/parcelamentos` | HandleCreateParcelamento |
| POST | `/parcelamentos/{id}/update` | HandleUpdateParcelamento |
| POST | `/parcelamentos/{id}/delete` | HandleDeleteParcelamento |
| GET | `/pagamentos` | HandlePagamentos |

### Receitas (`screen: "receitas"`)

| Metodo | Rota | Handler |
|--------|------|---------|
| GET | `/receitas` | HandleReceitas |
| POST | `/receitas` | HandleCreateReceita |
| POST | `/receitas/{id}/update` | HandleUpdateReceita |
| POST | `/receitas/{id}/delete` | HandleDeleteReceita |

### Investimentos (`screen: "investimentos"`)

| Metodo | Rota | Handler |
|--------|------|---------|
| GET | `/investimentos` | HandleInvestimentos |
| POST | `/investimentos` | HandleCreateInvestimento |
| POST | `/investimentos/{id}/update` | HandleUpdateInvestimento |
| POST | `/investimentos/{id}/delete` | HandleDeleteInvestimento |
| POST | `/reserva-em` | HandleAddReservaEM |
| POST | `/reserva-em/{id}/update` | HandleUpdateReservaEM |
| POST | `/reserva-em/{id}/delete` | HandleDeleteReservaEM |

### Emprestimos (`screen: "emprestimos"`)

| Metodo | Rota | Handler |
|--------|------|---------|
| GET | `/emprestimos` | HandleEmprestimos |
| POST | `/emprestimos` | HandleCreateEmprestimo |
| POST | `/emprestimos/{id}/delete` | HandleDeleteEmprestimo |
| POST | `/emprestimos/{id}/toggle-pago` | HandleToggleEmprestimoPago |
| POST | `/emprestimos/{id}/pagamento` | HandlePagamentoEmprestimo |

### Planejamento (`screen: "planejamento"`)

| Metodo | Rota | Handler |
|--------|------|---------|
| GET | `/planejamento` | HandlePlanejamento |

### Cadastros (`screen: "cadastros"`)

| Metodo | Rota | Handler |
|--------|------|---------|
| GET | `/cadastros` | HandleCadastros |
| POST | `/cadastros/categorias` | HandleCreateCategoria |
| POST | `/cadastros/categorias/{id}/update` | HandleUpdateCategoria |
| POST | `/cadastros/categorias/{id}/delete` | HandleDeleteCategoria |
| POST | `/cadastros/categorias/{id}/toggle-ativa` | HandleToggleCategoriaAtiva |
| POST | `/cadastros/cartoes` | HandleCreateCartao |
| POST | `/cadastros/cartoes/{id}/update` | HandleUpdateCartao |
| POST | `/cadastros/cartoes/{id}/delete` | HandleDeleteCartao |
| POST | `/cadastros/cartoes/{id}/toggle-ativo` | HandleToggleCartaoAtivo |

### Transacoes (`screen: "transacoes"`)

| Metodo | Rota | Handler |
|--------|------|---------|
| GET | `/transacoes` | HandleTransacoes |
| POST | `/transacoes/import-csv` | HandleImportCSV |
| POST | `/transacoes/import-ofx` | HandleImportOFX |
| POST | `/transacoes/{id}/converter` | HandleTransacaoConverter |
| POST | `/transacoes/{id}/ignorar` | HandleTransacaoIgnorar |
| POST | `/transacoes/{id}/categorizar` | HandleTransacaoCategorizar |

## AdminOnly (admin global)

### Pluggy Integration

| Metodo | Rota | Handler |
|--------|------|---------|
| POST | `/cadastros/pluggy/config` | HandleSavePluggyConfig |
| POST | `/cadastros/pluggy/connect-token` | HandlePluggyConnectToken |
| POST | `/cadastros/pluggy/items` | HandlePluggyRegisterItem |
| POST | `/cadastros/pluggy/{item_id}/disconnect` | HandlePluggyDisconnect |
| POST | `/cadastros/pluggy/{item_id}/sync` | HandlePluggySync |

### Usuarios

| Metodo | Rota | Handler |
|--------|------|---------|
| GET | `/usuarios` | HandleUsuarios |
| POST | `/usuarios` | HandleCreateUsuario |
| POST | `/usuarios/{id}/toggle-admin` | HandleToggleUsuarioAdmin |
| POST | `/usuarios/{id}/toggle-family-admin` | HandleToggleFamilyAdminFlag |
| POST | `/usuarios/{id}/toggle-ativo` | HandleToggleUsuarioAtivo |
| POST | `/usuarios/{id}/reset-senha` | HandleResetSenha |
| POST | `/usuarios/{id}/delete` | HandleDeleteUsuario |
| POST | `/usuarios/{id}/family` | HandleChangeUserFamily |

### Familias

| Metodo | Rota | Handler |
|--------|------|---------|
| GET | `/familias` | HandleFamilias |
| POST | `/familias` | HandleCreateFamilia |
| POST | `/familias/{id}/rename` | HandleRenameFamilia |
| POST | `/familias/{id}/delete` | HandleDeleteFamilia |

## FamilyAdminOnly (admin de familia)

| Metodo | Rota | Handler |
|--------|------|---------|
| GET | `/minha-familia` | HandleMeuTime |
| POST | `/minha-familia` | HandleMeuTimeCreateMembro |
| POST | `/minha-familia/{id}/toggle-ativo` | HandleMeuTimeToggleAtivo |
| POST | `/minha-familia/{id}/reset-senha` | HandleMeuTimeResetSenha |
| POST | `/minha-familia/{id}/permissoes/{tela}/toggle` | HandleToggleScreenPermission |

### Relatorios (ScreenProtected)

| Metodo | Rota | Handler | Screen |
|--------|------|---------|--------|
| GET | `/relatorio/despesas` | HandleRelatorioDespesas | despesas |
| GET | `/relatorio/receitas` | HandleRelatorioReceitas | receitas |
| GET | `/relatorio/investimentos` | HandleRelatorioInvestimentos | investimentos |
| GET | `/relatorio/emprestimos` | HandleRelatorioEmprestimos | emprestimos |
