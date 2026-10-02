package main

import (
	"log"
	"net/http"
	"os"

	"fincontrol/internal/auth"
	"fincontrol/internal/handler"
	"fincontrol/internal/integrations"
	"fincontrol/internal/store"
)

func getEnv(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

func main() {
	db := store.NewDB()
	defer db.Close()

	if err := store.RunMigrations(db); err != nil {
		log.Fatal("migrate:", err)
	}
	if err := store.EnsureAdmin(db); err != nil {
		log.Fatal("ensure admin:", err)
	}

	handler.InitTemplates()
	mux := http.NewServeMux()

	// Estáticos (sem auth)
	mux.Handle("GET /static/", http.StripPrefix("/static/", http.FileServer(http.Dir("static"))))

	// Login / Logout (sem auth)
	mux.HandleFunc("GET /login", handler.HandleLogin(db))
	mux.HandleFunc("POST /login", handler.HandleLogin(db))
	mux.HandleFunc("POST /logout", handler.HandleLogout(db))
	mux.HandleFunc("POST /auth/google/callback", handler.HandleGoogleCallback(db))
	mux.HandleFunc("GET /register", handler.HandleRegister(db))
	mux.HandleFunc("POST /register", handler.HandleRegister(db))
	mux.HandleFunc("GET /forgot-password", handler.HandleForgotPassword(db))
	mux.HandleFunc("POST /forgot-password", handler.HandleForgotPassword(db))
	mux.HandleFunc("GET /reset-senha", handler.HandleResetPassword(db))
	mux.HandleFunc("POST /reset-senha", handler.HandleResetPassword(db))

	// Dashboard
	mux.HandleFunc("GET /", auth.Protected(db, handler.HandleDashboard(db)))
	mux.HandleFunc("POST /api/despesas/{id}/toggle-pago", auth.Protected(db, handler.HandleToggleDespesaPago(db)))
	mux.HandleFunc("POST /api/dashboard/widgets", auth.Protected(db, handler.HandleSaveWidgets(db)))
	mux.HandleFunc("POST /api/parcelamentos/{id}/toggle-pago", auth.Protected(db, handler.HandleToggleParcelamentoPago(db)))

	// Despesas
	mux.HandleFunc("GET /despesas", auth.ScreenProtected(db, "despesas", handler.HandleDespesas(db)))
	mux.HandleFunc("POST /despesas", auth.ScreenProtected(db, "despesas", handler.HandleCreateDespesa(db)))
	mux.HandleFunc("POST /despesas/{id}/update", auth.ScreenProtected(db, "despesas", handler.HandleUpdateDespesa(db)))
	mux.HandleFunc("POST /despesas/{id}/delete", auth.ScreenProtected(db, "despesas", handler.HandleDeleteDespesa(db)))
	mux.HandleFunc("POST /despesas/{id}/toggle-ativa", auth.ScreenProtected(db, "despesas", handler.HandleToggleDespesaAtiva(db)))

	// Parcelamentos
	mux.HandleFunc("GET /parcelamentos", auth.ScreenProtected(db, "despesas", handler.HandleParcelamentos(db)))
	mux.HandleFunc("POST /parcelamentos", auth.ScreenProtected(db, "despesas", handler.HandleCreateParcelamento(db)))
	mux.HandleFunc("POST /parcelamentos/{id}/update", auth.ScreenProtected(db, "despesas", handler.HandleUpdateParcelamento(db)))
	mux.HandleFunc("POST /parcelamentos/{id}/delete", auth.ScreenProtected(db, "despesas", handler.HandleDeleteParcelamento(db)))
	mux.HandleFunc("POST /parcelamentos/{id}/antecipar", auth.ScreenProtected(db, "despesas", handler.HandleAnteciparParcela(db)))
	mux.HandleFunc("GET /api/parcelamentos/{id}/antecipar-preview", auth.ScreenProtected(db, "despesas", handler.HandleAnteciparPreview(db)))

	// Receitas
	mux.HandleFunc("GET /receitas", auth.ScreenProtected(db, "receitas", handler.HandleReceitas(db)))
	mux.HandleFunc("POST /receitas", auth.ScreenProtected(db, "receitas", handler.HandleCreateReceita(db)))
	mux.HandleFunc("POST /receitas/{id}/update", auth.ScreenProtected(db, "receitas", handler.HandleUpdateReceita(db)))
	mux.HandleFunc("POST /receitas/{id}/delete", auth.ScreenProtected(db, "receitas", handler.HandleDeleteReceita(db)))

	// Pagamentos do Mês
	mux.HandleFunc("GET /pagamentos", auth.ScreenProtected(db, "despesas", handler.HandlePagamentos(db)))

	// Investimentos
	mux.HandleFunc("GET /investimentos", auth.ScreenProtected(db, "investimentos", handler.HandleInvestimentos(db)))
	mux.HandleFunc("POST /investimentos", auth.ScreenProtected(db, "investimentos", handler.HandleCreateInvestimento(db)))
	mux.HandleFunc("POST /investimentos/{id}/update", auth.ScreenProtected(db, "investimentos", handler.HandleUpdateInvestimento(db)))
	mux.HandleFunc("POST /investimentos/{id}/delete", auth.ScreenProtected(db, "investimentos", handler.HandleDeleteInvestimento(db)))
	mux.HandleFunc("POST /reserva-em", auth.ScreenProtected(db, "investimentos", handler.HandleAddReservaEM(db)))
	mux.HandleFunc("POST /reserva-em/{id}/update", auth.ScreenProtected(db, "investimentos", handler.HandleUpdateReservaEM(db)))
	mux.HandleFunc("POST /reserva-em/{id}/delete", auth.ScreenProtected(db, "investimentos", handler.HandleDeleteReservaEM(db)))

	// Empréstimos
	mux.HandleFunc("GET /emprestimos", auth.ScreenProtected(db, "emprestimos", handler.HandleEmprestimos(db)))
	mux.HandleFunc("POST /emprestimos", auth.ScreenProtected(db, "emprestimos", handler.HandleCreateEmprestimo(db)))
	mux.HandleFunc("POST /emprestimos/{id}/delete", auth.ScreenProtected(db, "emprestimos", handler.HandleDeleteEmprestimo(db)))
	mux.HandleFunc("POST /emprestimos/{id}/toggle-pago", auth.ScreenProtected(db, "emprestimos", handler.HandleToggleEmprestimoPago(db)))
	mux.HandleFunc("POST /emprestimos/{id}/pagamento", auth.ScreenProtected(db, "emprestimos", handler.HandlePagamentoEmprestimo(db)))

	// Planejamento
	mux.HandleFunc("GET /planejamento", auth.ScreenProtected(db, "planejamento", handler.HandlePlanejamento(db)))

	// Cadastros
	mux.HandleFunc("GET /cadastros", auth.ScreenProtected(db, "cadastros", handler.HandleCadastros(db)))
	mux.HandleFunc("POST /cadastros/categorias", auth.ScreenProtected(db, "cadastros", handler.HandleCreateCategoria(db)))
	mux.HandleFunc("POST /cadastros/categorias/{id}/update", auth.ScreenProtected(db, "cadastros", handler.HandleUpdateCategoria(db)))
	mux.HandleFunc("POST /cadastros/categorias/{id}/delete", auth.ScreenProtected(db, "cadastros", handler.HandleDeleteCategoria(db)))
	mux.HandleFunc("POST /cadastros/categorias/{id}/toggle-ativa", auth.ScreenProtected(db, "cadastros", handler.HandleToggleCategoriaAtiva(db)))
	mux.HandleFunc("POST /cadastros/cartoes", auth.ScreenProtected(db, "cadastros", handler.HandleCreateCartao(db)))
	mux.HandleFunc("POST /cadastros/cartoes/{id}/update", auth.ScreenProtected(db, "cadastros", handler.HandleUpdateCartao(db)))
	mux.HandleFunc("POST /cadastros/cartoes/{id}/delete", auth.ScreenProtected(db, "cadastros", handler.HandleDeleteCartao(db)))
	mux.HandleFunc("POST /cadastros/cartoes/{id}/toggle-ativo", auth.ScreenProtected(db, "cadastros", handler.HandleToggleCartaoAtivo(db)))

	// Pluggy integration (admin only)
	mux.HandleFunc("POST /integracoes/pluggy/config", auth.AdminOnly(db, handler.HandleSavePluggyConfig(db)))
	mux.HandleFunc("POST /integracoes/pluggy/connect-token", auth.AdminOnly(db, handler.HandlePluggyConnectToken(db)))
	mux.HandleFunc("POST /integracoes/pluggy/items", auth.AdminOnly(db, handler.HandlePluggyRegisterItem(db)))
	mux.HandleFunc("POST /integracoes/pluggy/{item_id}/disconnect", auth.AdminOnly(db, handler.HandlePluggyDisconnect(db)))
	mux.HandleFunc("POST /integracoes/pluggy/{item_id}/sync", auth.AdminOnly(db, handler.HandlePluggySync(db)))

	// Pluggy webhook (público — validação por token no pluggy-service)
	mux.HandleFunc("POST /webhook/pluggy", handler.HandlePluggyWebhook(db))

	// Usuários (admin only)
	mux.HandleFunc("GET /usuarios", auth.AdminOnly(db, handler.HandleUsuarios(db)))
	mux.HandleFunc("POST /usuarios", auth.AdminOnly(db, handler.HandleCreateUsuario(db)))
	mux.HandleFunc("POST /usuarios/{id}/toggle-admin", auth.AdminOnly(db, handler.HandleToggleUsuarioAdmin(db)))
	mux.HandleFunc("POST /usuarios/{id}/toggle-family-admin", auth.AdminOnly(db, handler.HandleToggleFamilyAdminFlag(db)))
	mux.HandleFunc("POST /usuarios/{id}/toggle-ativo", auth.AdminOnly(db, handler.HandleToggleUsuarioAtivo(db)))
	mux.HandleFunc("POST /usuarios/{id}/reset-senha", auth.AdminOnly(db, handler.HandleResetSenha(db)))
	mux.HandleFunc("POST /usuarios/{id}/delete", auth.AdminOnly(db, handler.HandleDeleteUsuario(db)))
	mux.HandleFunc("POST /usuarios/{id}/family", auth.AdminOnly(db, handler.HandleChangeUserFamily(db)))

	// Minha Família (family admin)
	mux.HandleFunc("GET /minha-familia", auth.FamilyAdminOnly(db, handler.HandleMeuTime(db)))
	mux.HandleFunc("POST /minha-familia", auth.FamilyAdminOnly(db, handler.HandleMeuTimeCreateMembro(db)))
	mux.HandleFunc("POST /minha-familia/{id}/toggle-ativo", auth.FamilyAdminOnly(db, handler.HandleMeuTimeToggleAtivo(db)))
	mux.HandleFunc("POST /minha-familia/{id}/reset-senha", auth.FamilyAdminOnly(db, handler.HandleMeuTimeResetSenha(db)))
	mux.HandleFunc("POST /minha-senha", auth.Protected(db, handler.HandleMinhaSenha(db)))
	mux.HandleFunc("POST /minha-familia/{id}/permissoes/{tela}/toggle", auth.FamilyAdminOnly(db, handler.HandleToggleScreenPermission(db)))

	// Transações Bancárias
	mux.HandleFunc("GET /transacoes", auth.ScreenProtected(db, "transacoes", handler.HandleTransacoes(db)))
	mux.HandleFunc("POST /transacoes/import-csv", auth.ScreenProtected(db, "transacoes", handler.HandleImportCSV(db)))
	mux.HandleFunc("POST /transacoes/import-ofx", auth.ScreenProtected(db, "transacoes", handler.HandleImportOFX(db)))
	mux.HandleFunc("POST /transacoes/{id}/converter", auth.ScreenProtected(db, "transacoes", handler.HandleTransacaoConverter(db)))
	mux.HandleFunc("POST /transacoes/{id}/ignorar", auth.ScreenProtected(db, "transacoes", handler.HandleTransacaoIgnorar(db)))
	mux.HandleFunc("POST /transacoes/{id}/categorizar", auth.ScreenProtected(db, "transacoes", handler.HandleTransacaoCategorizar(db)))
	mux.HandleFunc("POST /transacoes/{id}/vincular", auth.ScreenProtected(db, "transacoes", handler.HandleTransacaoVincular(db)))
	mux.HandleFunc("POST /transacoes/{id}/desvincular", auth.ScreenProtected(db, "transacoes", handler.HandleTransacaoDesvincular(db)))
	mux.HandleFunc("GET /api/transacoes/{id}/matches", auth.ScreenProtected(db, "transacoes", handler.HandleTransacaoMatches(db)))
	mux.HandleFunc("POST /transacoes/auto-match", auth.ScreenProtected(db, "transacoes", handler.HandleAutoMatch(db)))
	mux.HandleFunc("POST /transacoes/ignorar-todas", auth.ScreenProtected(db, "transacoes", handler.HandleIgnorarTodas(db)))
	mux.HandleFunc("POST /transacoes/converter-todas", auth.ScreenProtected(db, "transacoes", handler.HandleConverterTodas(db)))

	// Importar / Exportar
	mux.HandleFunc("GET /dados", auth.Protected(db, handler.HandleImportPage(db)))
	mux.HandleFunc("GET /dados/modelo", auth.Protected(db, handler.HandleExportTemplate(db)))
	mux.HandleFunc("GET /dados/exportar", auth.Protected(db, handler.HandleExport(db)))
	mux.HandleFunc("POST /dados/importar", auth.Protected(db, handler.HandleImport(db)))

	// Relatórios
	mux.HandleFunc("GET /relatorio/dashboard", auth.Protected(db, handler.HandleRelatorioDashboard(db)))
	mux.HandleFunc("GET /relatorio/despesas", auth.ScreenProtected(db, "despesas", handler.HandleRelatorioDespesas(db)))
	mux.HandleFunc("GET /relatorio/receitas", auth.ScreenProtected(db, "receitas", handler.HandleRelatorioReceitas(db)))
	mux.HandleFunc("GET /relatorio/investimentos", auth.ScreenProtected(db, "investimentos", handler.HandleRelatorioInvestimentos(db)))
	mux.HandleFunc("GET /relatorio/emprestimos", auth.ScreenProtected(db, "emprestimos", handler.HandleRelatorioEmprestimos(db)))

	// Famílias (admin only)
	mux.HandleFunc("GET /familias", auth.AdminOnly(db, handler.HandleFamilias(db)))
	mux.HandleFunc("POST /familias", auth.AdminOnly(db, handler.HandleCreateFamilia(db)))
	mux.HandleFunc("POST /familias/{id}/rename", auth.AdminOnly(db, handler.HandleRenameFamilia(db)))
	mux.HandleFunc("POST /familias/{id}/delete", auth.AdminOnly(db, handler.HandleDeleteFamilia(db)))

	// Integrações
	mux.HandleFunc("GET /integracoes", auth.Protected(db, handler.HandleIntegracoes(db)))
	mux.HandleFunc("POST /integracoes/{nome}", auth.Protected(db, handler.HandleIntegracaoSalvar(db)))
	mux.HandleFunc("POST /integracoes/{nome}/testar", auth.Protected(db, handler.HandleIntegracaoTestar(db)))

	// Iniciar scheduler de integrações
	integrations.StartScheduler(db)

	port := getEnv("PORT", "8080")
	log.Printf("Servidor iniciado em http://localhost:%s", port)
	if err := http.ListenAndServe(":"+port, mux); err != nil {
		log.Fatal(err)
	}
}
