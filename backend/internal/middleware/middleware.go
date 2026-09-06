// Continuação do package middleware.
package middleware

import (
	"context" // para usar context.WithValue
	"fmt"      // debug prints
	"net/http"
)

// WithJWTSecret injeta o segredo JWT no contexto da request.
// Outros middlewares/handlers leem via r.Context().Value("jwt_secret").
//
// Em Go, a chave de context.Value idealmente é um tipo customizado
// para evitar colisões. Aqui está como string por simplicidade (MVP).
func WithJWTSecret(secret string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			fmt.Println("[MIDDLEWARE] WithJWTSecret injetando jwt_secret no contexto")
			ctx := r.Context()
			ctx = context.WithValue(ctx, "jwt_secret", secret)
			// Importante: passar r.WithContext(ctx), não o r original,
			// senão o valor some.
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}
