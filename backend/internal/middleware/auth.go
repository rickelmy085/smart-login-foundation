// Package middleware contém interceptores de request.
// Em Go, um middleware é uma função que recebe um http.Handler
// e devolve outro http.Handler — "embrulhando" o original.
package middleware

import (
	"context"
	"encoding/json"
	"fmt" // debug prints
	"net/http"
	"strings"

	"github.com/bsmart/abis/internal/service"
)

// AuthMiddleware bloqueia requests sem token JWT válido.
// Se válido, injeta o employee_id no contexto e passa adiante.
func AuthMiddleware(auth *service.AuthService) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			fmt.Println("[MIDDLEWARE] AuthMiddleware iniciado")
			secret := r.Context().Value("jwt_secret").(string)
			token := bearerToken(r)
			if token == "" {
				fmt.Println("[MIDDLEWARE] AuthMiddleware token ausente")
				writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "missing token"})
				return
			}
			fmt.Printf("[MIDDLEWARE] AuthMiddleware token recebido length=%d\n", len(token))

			res, err := auth.Me(r.Context(), token, []byte(secret))
			if err != nil {
				fmt.Printf("[MIDDLEWARE] AuthMiddleware token invalido: %v\n", err)
				writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "invalid or expired session"})
				return
			}
			fmt.Printf("[MIDDLEWARE] AuthMiddleware token valido employeeID=%s\n", res.Employee.ID)

			ctx := r.Context()
			ctx = context.WithValue(ctx, "employee_id", res.Employee.ID)
			next.ServeHTTP(w, r.WithContext(ctx))
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
