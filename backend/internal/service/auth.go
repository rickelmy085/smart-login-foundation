// Package service contém as regras de negócio.
// Aqui NÃO se fala HTTP nem SQL direto: depende de repository (dados)
// e expõe métodos de "caso de uso" (Login, Me).
package service

import (
	"context"
	"errors" // para criar erros sentinela com errors.New
	"log/slog"
	"time"

	"github.com/bsmart/abis/internal/models"
	"github.com/bsmart/abis/internal/repository"
	"github.com/golang-jwt/jwt/v5" // biblioteca JWT
	"github.com/google/uuid"        // geração de UUID v4
)

// Constantes de tempo de vida do token/sessão.
const (
	tokenTTLRemember = 7 * 24 * time.Hour // 7 dias (checkbox "manter conectado")
	tokenTTLDefault  = 12 * time.Hour     // sessão padrão
)

// AuthService concentra toda a lógica de autenticação.
// Recebe repositórios por injeção de dependência (facilita testes).
type AuthService struct {
	employees *repository.EmployeeRepo
	sessions  *repository.SessionRepo
	ttl       time.Duration                // TTL padrão
	now       func() time.Time             // injetável: facilita testes com relógio fixo
}

// NewAuthService monta o service. ttl <= 0 usa o padrão (12h).
func NewAuthService(employees *repository.EmployeeRepo, sessions *repository.SessionRepo, ttl time.Duration) *AuthService {
	if ttl <= 0 {
		ttl = tokenTTLDefault
	}
	return &AuthService{
		employees: employees,
		sessions:  sessions,
		ttl:       ttl,
		now:       time.Now, // função padrão; substituível em testes
	}
}

// Login valida credenciais e, em caso de sucesso, cria sessão + emite JWT.
//
// Fluxo:
//  1) Busca employee pelo RE.
//  2) Compara senha (texto puro, decisão de MVP).
//  3) Cria registro de sessão no banco.
//  4) Gera JWT contendo employee_id, session_id, etc.
//  5) Devolve LoginResponse com token + employee + sessão.
func (s *AuthService) Login(ctx context.Context, secret []byte, req models.LoginRequest) (models.LoginResponse, error) {
	emp, err := s.employees.FindByRE(ctx, req.RE)
	if err != nil {
		slog.Info("login: FindByRE failed", "re", req.RE, "error", err.Error())
		return models.LoginResponse{}, ErrInvalidCredentials
	}

	slog.Info("login attempt", "re", req.RE)

	// Comparação direta de strings. NÃO recomendado em produção (use bcrypt/argon2).
	if emp.Password != req.Password {
		slog.Info("login failed", "re", req.RE)
		return models.LoginResponse{}, ErrInvalidCredentials
	}

	// Calcula expiração: 7 dias se "remember", senão ttl padrão.
	expiresAt := s.now().Add(s.ttl)
	if req.Remember {
		expiresAt = s.now().Add(tokenTTLRemember)
	}

	// Cria sessão nova no banco.
	sessionID := uuid.NewString()
	session := models.Session{
		ID:         sessionID,
		EmployeeID: emp.ID,
		// Formato igual ao DEFAULT datetime('now') do SQLite, pra ficar uniforme.
		ExpiresAt: expiresAt.UTC().Format("2006-01-02 15:04:05"),
	}

	if err := s.sessions.Create(ctx, session); err != nil {
		return models.LoginResponse{}, err
	}

	// Assina o JWT.
	token, err := s.buildToken(secret, emp, session)
	if err != nil {
		return models.LoginResponse{}, err
	}

	return models.LoginResponse{
		Token:    token,
		Session:  session,
		Employee: emp,
	}, nil
}

// Me valida um token JWT e devolve o employee correspondente.
// Usado em todo request autenticado (incluindo pelo middleware).
func (s *AuthService) Me(ctx context.Context, tokenString string, secret []byte) (models.MeResponse, error) {
	// Parse + valida assinatura e expiração do JWT.
	token, err := parseToken(tokenString, secret)
	if err != nil {
		return models.MeResponse{}, err
	}

	// Claims são os "dados" dentro do JWT (payload).
	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok || !token.Valid {
		return models.MeResponse{}, ErrUnauthorized
	}

	sessionID, _ := claims["session_id"].(string)
	employeeID, _ := claims["employee_id"].(string)
	if sessionID == "" || employeeID == "" {
		return models.MeResponse{}, ErrUnauthorized
	}

	// Confere que a sessão ainda existe no banco e pertence ao employee certo.
	session, err := s.sessions.FindByID(ctx, sessionID)
	if err != nil || session.EmployeeID != employeeID {
		return models.MeResponse{}, ErrUnauthorized
	}
	// Parse da data em texto e checa expiração.
	expiresAt, perr := time.Parse("2006-01-02 15:04:05", session.ExpiresAt)
	if perr != nil || !expiresAt.After(s.now()) {
		return models.MeResponse{}, ErrUnauthorized
	}

	emp, err := s.employees.FindByID(ctx, employeeID)
	if err != nil {
		return models.MeResponse{}, ErrUnauthorized
	}

	return models.MeResponse{Employee: emp}, nil
}

// buildToken monta e assina o JWT.
// Claims são os pares chave/valor que ficam dentro do token (payload).
func (s *AuthService) buildToken(secret []byte, emp models.Employee, session models.Session) (string, error) {
	now := s.now()
	claims := jwt.MapClaims{
		"employee_id": emp.ID,
		"re":          emp.RE,
		"name":        emp.Name,
		"role":        emp.Role,
		"session_id":  session.ID,
		"iat":         now.Unix(),                  // issued at
		"exp":         now.Add(s.ttl).Unix(),       // expiration
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(secret) // assina com HMAC-SHA256
}

// parseToken valida o JWT e devolve o token decodificado.
// O keyFunc garante que o método de assinatura é HMAC (evita truques tipo "none").
func parseToken(tokenString string, secret []byte) (*jwt.Token, error) {
	return jwt.Parse(tokenString, func(t *jwt.Token) (interface{}, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, ErrUnauthorized
		}
		return secret, nil
	})
}

// Erros "sentinela": usados com errors.Is no handler para mapear status HTTP.
// Por convenção, em Go são exportados e em CamelCase começando com Err.
var (
	ErrInvalidCredentials = errors.New("invalid credentials")
	ErrUnauthorized       = errors.New("unauthorized")
)
