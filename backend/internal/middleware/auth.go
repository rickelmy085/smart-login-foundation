// Package middleware contém interceptores de request.
// Em Go, um middleware é uma função que recebe um http.Handler
// e devolve outro http.Handler — "embrulhando" o original.
package middleware

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/bsmart/abis/internal/service"
)

// AuthMiddleware bloqueia requests sem token JWT válido.
// Se válido, passa adiante (next.ServeHTTP).
// Caso contrário, devolve 401 direto.
//
// Recebe o *service.AuthService para reusar a lógica de validação.
func AuthMiddleware(auth *service.AuthService) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Pega o segredo JWT (injetado por WithJWTSecret mais acima na pilha).
			secret := r.Context().Value("jwt_secret").(string)
			token := bearerToken(r)
			if token == "" {
				writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "missing token"})
				return
			}

			// Reusa o Me() do service: ele já valida assinatura, sessão e expiração.
			if _, err := auth.Me(r.Context(), token, []byte(secret)); err != nil {
				writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "invalid or expired session"})
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

// bearerToken extrai o token de "Authorization: Bearer XYZ".
func bearerToken(r *http.Request) string {
	header := r.Header.Get("Authorization")
	if header == "" {
		return ""
	}
	parts := strings.SplitN(header, " ", 2)
	if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
		return ""
	}
	return strings.TrimSpace(parts[1])
}

// writeJSON envia resposta JSON. (Duplicado do handler — em projetos
// maiores, vale extrair para um pacote interno "httpx" comum.)
func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}
