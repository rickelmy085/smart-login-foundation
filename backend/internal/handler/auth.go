// Package handler é a camada HTTP: lê a request, chama o service,
// serializa a resposta em JSON. Não toca no banco nem em regras.
package handler

import (
	"encoding/json" // marshal/unmarshal JSON
	"fmt"            // debug prints
	"net/http"      // tipos do servidor HTTP
	"strings"       // manipulação de strings (TrimSpace, SplitN)

	"github.com/bsmart/abis/internal/middleware"
	"github.com/bsmart/abis/internal/models"
	"github.com/bsmart/abis/internal/service"
	"github.com/go-chi/chi/v5" // roteador chi (atualmente NÃO usado pelo main.go, mas mantido para evoluções)
)

// AuthHandler agrupa os endpoints de autenticação.
// Recebe o AuthService por injeção (dependência).
type AuthHandler struct {
	auth *service.AuthService
}

// Construtor idiomático: New + tipo + ponteiro de retorno.
func NewAuthHandler(auth *service.AuthService) *AuthHandler {
	return &AuthHandler{auth: auth}
}

// Routes define as rotas via chi. (main.go hoje usa http.ServeMux,
// então este método existe para uma migração futura.)
func (h *AuthHandler) Routes() chi.Router {
	r := chi.NewRouter()
	r.Post("/login", h.Login)
	// With() aplica middlewares SÓ para esta rota.
	r.With(middleware.AuthMiddleware(h.auth)).Get("/me", h.Me)
	r.Post("/logout", h.Logout)
	return r
}

// Login (POST /api/login) decodifica o corpo JSON e chama o service.
// Responde 200 com o LoginResponse ou 401/400 conforme o erro.
func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	fmt.Println("[AUTH] Login handler iniciado")
	var req models.LoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		fmt.Printf("[AUTH] Login erro decode JSON: %v\n", err)
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	// TrimSpace remove espaços acidentais no RE.
	req.RE = strings.TrimSpace(req.RE)
	if req.RE == "" || req.Password == "" {
		fmt.Println("[AUTH] Login RE ou password vazios")
		writeError(w, http.StatusBadRequest, "re and password are required")
		return
	}
	fmt.Printf("[AUTH] Login tentativa RE=%s remember=%v\n", req.RE, req.Remember)

	// Pega o segredo JWT injetado pelo middleware WithJWTSecret.
	res, err := h.auth.Login(r.Context(), []byte(r.Context().Value("jwt_secret").(string)), req)
	if err != nil {
		fmt.Printf("[AUTH] Login falhou RE=%s erro=%v\n", req.RE, err)
		switch {
		case err == service.ErrInvalidCredentials:
			writeError(w, http.StatusUnauthorized, "RE ou senha inválidos")
		default:
			writeError(w, http.StatusInternalServerError, "internal error")
		}
		return
	}
	fmt.Printf("[AUTH] Login sucesso RE=%s token gerado\n", req.RE)

	writeJSON(w, http.StatusOK, res)
}

// Me (GET /api/me) exige Authorization: Bearer <token> e devolve o employee.
// O middleware AuthMiddleware já protege essa rota, mas Me também
// funciona isolado quando chamado diretamente (ex: o próprio middleware).
func (h *AuthHandler) Me(w http.ResponseWriter, r *http.Request) {
	fmt.Println("[AUTH] Me handler iniciado")
	secret := r.Context().Value("jwt_secret").(string)
	token := bearerToken(r)
	if token == "" {
		fmt.Println("[AUTH] Me token ausente")
		writeError(w, http.StatusUnauthorized, "missing token")
		return
	}
	fmt.Printf("[AUTH] Me token recebido length=%d\n", len(token))

	res, err := h.auth.Me(r.Context(), token, []byte(secret))
	if err != nil {
		fmt.Printf("[AUTH] Me falhou: %v\n", err)
		writeError(w, http.StatusUnauthorized, "invalid or expired session")
		return
	}
	fmt.Printf("[AUTH] Me sucesso employeeID=%s name=%s\n", res.Employee.ID, res.Employee.Name)

	writeJSON(w, http.StatusOK, res)
}

// Logout (POST /api/logout) — placeholder. Uma versão completa
// removeria o registro de sessão do banco via sessionRepo.Delete.
func (h *AuthHandler) Logout(w http.ResponseWriter, r *http.Request) {
	fmt.Println("[AUTH] Logout chamado")
	writeJSON(w, http.StatusOK, map[string]string{"message": "logged out"})
}

// bearerToken extrai o token do header "Authorization: Bearer XYZ".
func bearerToken(r *http.Request) string {
	header := r.Header.Get("Authorization")
	if header == "" {
		return ""
	}
	parts := strings.SplitN(header, " ", 2) // divide em ["Bearer", "XYZ..."]
	if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
		return ""
	}
	return strings.TrimSpace(parts[1])
}

// writeJSON serializa `v` como JSON e envia com o status informado.
// Usar uma helper evita repetir Content-Type/WriteHeader em todo endpoint.
func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

// writeError envia um JSON padronizado { "error": "..." }.
func writeError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, map[string]string{"error": message})
}
