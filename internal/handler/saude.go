package handler

import (
	"context"
	"database/sql"
	"log"
	"net/http"
	"time"
)

// tempoMaximoParaResponderAoBanco limita a espera do ping: o healthcheck do
// Docker precisa de resposta rapida, mesmo com o banco travado.
const tempoMaximoParaResponderAoBanco = 2 * time.Second

// HandleVerificarSaude responde 200 quando o app consegue falar com o banco e
// 503 quando nao consegue. Usado pelo healthcheck do Docker e pelo deploy.
// Rota publica: a resposta nao revela nenhum detalhe do erro.
func HandleVerificarSaude(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		w.Header().Set("Cache-Control", "no-store")

		contexto, cancelar := context.WithTimeout(r.Context(), tempoMaximoParaResponderAoBanco)
		defer cancelar()

		if err := db.PingContext(contexto); err != nil {
			log.Printf("healthz: banco indisponivel: %v", err)
			w.WriteHeader(http.StatusServiceUnavailable)
			w.Write([]byte("banco indisponivel\n"))
			return
		}
		w.Write([]byte("ok\n"))
	}
}
