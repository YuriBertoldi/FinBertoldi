package main

import (
	"database/sql"
	"log"
	"net/http"
)

type App struct {
	db *sql.DB
}

func main() {
	db := newDB()
	defer db.Close()

	if err := dbAutoMigrate(db); err != nil {
		log.Fatal("migrate:", err)
	}
	if err := dbEnsureAdmin(db); err != nil {
		log.Fatal("ensure admin:", err)
	}

	app := &App{db: db}
	initTemplates()

	mux := http.NewServeMux()

	// Arquivos estáticos (sem auth)
	mux.Handle("GET /static/", http.StripPrefix("/static/", http.FileServer(http.Dir("static"))))

	// Login / Logout (sem auth)
	mux.HandleFunc("GET /login", handleLogin(app))
	mux.HandleFunc("POST /login", handleLogin(app))
	mux.HandleFunc("POST /logout", handleLogout(app))

	// Dashboard
	mux.HandleFunc("GET /", protected(app, handleDashboard(app)))
	mux.HandleFunc("POST /api/despesas/{id}/toggle-pago", protected(app, handleToggleDespesaPago(app)))

	// Despesas
	mux.HandleFunc("GET /despesas", protected(app, handleDespesas(app)))
	mux.HandleFunc("POST /despesas", protected(app, handleCreateDespesa(app)))
	mux.HandleFunc("POST /despesas/{id}/update", protected(app, handleUpdateDespesa(app)))
	mux.HandleFunc("POST /despesas/{id}/delete", protected(app, handleDeleteDespesa(app)))
	mux.HandleFunc("POST /despesas/{id}/toggle-ativa", protected(app, handleToggleDespesaAtiva(app)))

	// Parcelamentos
	mux.HandleFunc("GET /parcelamentos", protected(app, handleParcelamentos(app)))
	mux.HandleFunc("POST /parcelamentos", protected(app, handleCreateParcelamento(app)))
	mux.HandleFunc("POST /parcelamentos/{id}/update", protected(app, handleUpdateParcelamento(app)))
	mux.HandleFunc("POST /parcelamentos/{id}/delete", protected(app, handleDeleteParcelamento(app)))

	// Receitas
	mux.HandleFunc("GET /receitas", protected(app, handleReceitas(app)))
	mux.HandleFunc("POST /receitas", protected(app, handleCreateReceita(app)))
	mux.HandleFunc("POST /receitas/{id}/delete", protected(app, handleDeleteReceita(app)))

	// Investimentos
	mux.HandleFunc("GET /investimentos", protected(app, handleInvestimentos(app)))
	mux.HandleFunc("POST /investimentos", protected(app, handleCreateInvestimento(app)))
	mux.HandleFunc("POST /investimentos/{id}/delete", protected(app, handleDeleteInvestimento(app)))
	mux.HandleFunc("POST /reserva-em", protected(app, handleAddReservaEM(app)))
	mux.HandleFunc("POST /reserva-em/{id}/delete", protected(app, handleDeleteReservaEM(app)))

	// Empréstimos
	mux.HandleFunc("GET /emprestimos", protected(app, handleEmprestimos(app)))
	mux.HandleFunc("POST /emprestimos", protected(app, handleCreateEmprestimo(app)))
	mux.HandleFunc("POST /emprestimos/{id}/delete", protected(app, handleDeleteEmprestimo(app)))
	mux.HandleFunc("POST /api/emprestimos/{id}/toggle-pago", protected(app, handleToggleEmprestimoPago(app)))

	// Planejamento
	mux.HandleFunc("GET /planejamento", protected(app, handlePlanejamento(app)))

	// Usuários (admin only)
	mux.HandleFunc("GET /usuarios", handleUsuarios(app))
	mux.HandleFunc("POST /usuarios", handleCreateUsuario(app))
	mux.HandleFunc("POST /usuarios/{id}/toggle-admin", handleToggleUsuarioAdmin(app))
	mux.HandleFunc("POST /usuarios/{id}/toggle-ativo", handleToggleUsuarioAtivo(app))
	mux.HandleFunc("POST /usuarios/{id}/reset-senha", handleResetSenha(app))
	mux.HandleFunc("POST /usuarios/{id}/delete", handleDeleteUsuario(app))

	port := getEnv("PORT", "8080")
	log.Printf("Servidor iniciado em http://localhost:%s", port)
	if err := http.ListenAndServe(":"+port, mux); err != nil {
		log.Fatal(err)
	}
}
