// Pacote principal (executável). Em Go, todo programa começa em package main
// com uma função chamada main().
package main

import (
	// Pacotes da biblioteca padrão:
	"context"   // Usado para passar deadlines/cancelamento para o banco de dados.
	"log/slog"  // Logger estruturado moderno do Go (substitui o "log" antigo).
	"net/http"  // Servidor HTTP e tipos Request/Response/Handler.
	"os"        // Funções do sistema operacional (variáveis de ambiente, Exit, etc.).

	// Pacotes internos deste projeto. Cada um mora na pasta correspondente:
	// config     = configurações (porta, segredo JWT, caminho do DB).
	// database   = abre a conexão com o SQLite e roda migrações.
	// handler    = camada HTTP que responde às requisições.
	// middleware = interceptores de request (autenticação, logs).
	// repository = camada de acesso ao banco (queries SQL).
	// service    = regras de negócio (login, validação de sessão, etc.).
	"time"

	"github.com/bsmart/abis/internal/config"
	"github.com/bsmart/abis/internal/database"
	"github.com/bsmart/abis/internal/handler"
	"github.com/bsmart/abis/internal/middleware"
	"github.com/bsmart/abis/internal/repository"
	"github.com/bsmart/abis/internal/service"
)

// Função principal: o servidor é montado e iniciado aqui.
func main() {
	// 1) Lê configurações do ambiente (PORT, JWT_SECRET, DATABASE_PATH).
	cfg := config.Load()

	// 2) Abre conexão com o SQLite. Se falhar, aborta.
	db, err := database.Connect(cfg.DatabasePath)
	if err != nil {
		slog.Error("connect database", "error", err) // log estruturado
		os.Exit(1)                                   // sai com código 1 = erro
	}
	// Garante que o banco será fechado quando main() terminar.
	defer db.Close()

	// 3) Cria as tabelas (se não existirem) e popula o usuário demo.
	if err := database.Migrate(context.Background(), db); err != nil {
		slog.Error("run migrations", "error", err)
		os.Exit(1)
	}

	// 4) Monta as camadas em ordem (de dentro pra fora):
	//    repo → service → handler → rotas HTTP.
	employeeRepo := repository.NewEmployeeRepo(db)        // SQL de employees
	sessionRepo := repository.NewSessionRepo(db)          // SQL de sessions
	authService := service.NewAuthService(employeeRepo, sessionRepo, 0) // regras de auth
	authHandler := handler.NewAuthHandler(authService)    // expõe endpoints HTTP

	// 5) Roteador HTTP (mux = multiplexador de rotas).
	mux := http.NewServeMux()
	// Cada rota aponta para um método do handler.
	mux.HandleFunc("/api/login", authHandler.Login)
	mux.HandleFunc("/api/me", authHandler.Me)
	mux.HandleFunc("/api/logout", authHandler.Logout)

	// 6) Empilha os middlewares. A ordem importa: o que está mais
	//    à esquerda envolve o que está à direita.
	//    Fluxo: request → CORS → log → injeta JWT secret → mux (rotas).
	addr := ":" + cfg.Port
	slog.Info("server started", "addr", addr)
	if err := http.ListenAndServe(addr,
		corsMiddleware(
			loggingMiddleware(
				middleware.WithJWTSecret(cfg.JWTSecret)(mux),
			),
		),
	); err != nil {
		slog.Error("server stopped", "error", err)
	}
}

// Middleware de CORS (Cross-Origin Resource Sharing).
// Sem ele, o front-end em http://localhost:8080 não conseguiria chamar
// a API em http://localhost:8081 (bloqueio do navegador).
func corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Cabeçalhos HTTP que o navegador precisa ver.
		w.Header().Set("Access-Control-Allow-Origin", "http://localhost:8080")
		w.Header().Set("Access-Control-Allow-Methods", "POST, GET, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")

		// Preflight request: o navegador envia OPTIONS antes de POST/GET
		// com headers customizados. Respondemos OK e paramos.
		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}
		// Segue para o próximo handler/middleware.
		next.ServeHTTP(w, r)
	})
}

// Middleware que loga cada requisição com método, caminho e duração.
func loggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		next.ServeHTTP(w, r) // deixa a request passar
		slog.Info("request",
			"method", r.Method,
			"path", r.URL.Path,
			"duration", time.Since(start),
		)
	})
}
