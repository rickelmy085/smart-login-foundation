package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"time"

	"github.com/bsmart/abis/internal/chat"
	"github.com/bsmart/abis/internal/config"
	"github.com/bsmart/abis/internal/database"
	"github.com/bsmart/abis/internal/documentengine"
	"github.com/bsmart/abis/internal/groq"
	"github.com/bsmart/abis/internal/handler"
	"github.com/bsmart/abis/internal/knowledge"
	"github.com/bsmart/abis/internal/metrics"
	"github.com/bsmart/abis/internal/middleware"
	"github.com/bsmart/abis/internal/repository"
	"github.com/bsmart/abis/internal/service"
	"github.com/bsmart/abis/internal/web"
)

func main() {
	fmt.Println("[MAIN] Iniciando servidor ABIS")

	if err := config.LoadEnvFile(".env"); err != nil {
		fmt.Printf("[MAIN] Aviso ao carregar .env: %v\n", err)
	}

	cfg := config.Load()
	fmt.Printf("[MAIN] Config carregada: porta=%s db=%s frontend_urls=%v\n", cfg.Port, cfg.DatabasePath, cfg.FrontendURLs)

	db, err := database.Connect(cfg.DatabasePath)
	if err != nil {
		fmt.Printf("[MAIN] Erro ao conectar banco: %v\n", err)
		slog.Error("connect database", "error", err)
		os.Exit(1)
	}
	defer db.Close()
	fmt.Println("[MAIN] Banco conectado com sucesso")

	ctx := context.Background()
	for _, m := range []struct {
		name string
		fn   func(context.Context, *sql.DB) error
	}{
		{"Migrate", database.Migrate},
		{"MigrateKnowledge", database.MigrateKnowledge},
		{"MigrateWorkflow", database.MigrateWorkflow},
		{"SeedWorkflowTemplates", database.SeedWorkflowTemplates},
	} {
		if err := m.fn(ctx, db); err != nil {
			fmt.Printf("[MAIN] Erro em %s: %v\n", m.name, err)
			slog.Error(m.name, "error", err)
			os.Exit(1)
		}
		fmt.Printf("[MAIN] %s executada com sucesso\n", m.name)
	}

	employeeRepo := repository.NewEmployeeRepo(db)
	sessionRepo := repository.NewSessionRepo(db)
	workflowRepo := repository.NewWorkflowRepo(db)
	chatRepo := repository.NewChatRepo(db)

	authService := service.NewAuthService(employeeRepo, sessionRepo, 0)
	groqClient := groq.New(cfg.GroqAPIKey, cfg.GroqModel)
	searcher := knowledge.NewSearcher(db)
	webClient := web.NewClient()
	chatService := chat.NewService(searcher, groqClient, webClient)
	workflowService := service.NewWorkflowService(workflowRepo, searcher, groqClient)

	// Configure Document Engine if enabled
	if cfg.DocumentEngineEnabled {
		docEngineClient := documentengine.NewClient(
			cfg.DocumentEngineURL,
			cfg.DocumentEngineSecret,
			time.Duration(cfg.DocumentEngineTimeout)*time.Second,
		)
		workflowService.WithDocumentEngine(docEngineClient, true, cfg.DocumentEngineFallback)
		workflowService.WithEmployeeRepo(employeeRepo)
		fmt.Printf("[MAIN] Document Engine habilitado: url=%s timeout=%ds fallback=%v\n",
			cfg.DocumentEngineURL, cfg.DocumentEngineTimeout, cfg.DocumentEngineFallback)
	} else {
		fmt.Println("[MAIN] Document Engine desabilitado usando gerador Go")
	}

	authHandler := handler.NewAuthHandler(authService, sessionRepo)
	chatHandler := handler.NewChatHandler(chatService, workflowService, employeeRepo, chatRepo)
	knowledgeHandler := handler.NewKnowledgeHandler(searcher)
	workflowHandler := handler.NewWorkflowHandler(workflowService, workflowRepo)

	mux := http.NewServeMux()
	mux.HandleFunc("/api/health", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
	})

	// Metrics endpoint
	mux.HandleFunc("/api/metrics", func(w http.ResponseWriter, r *http.Request) {
		snapshot := metrics.GlobalMetrics.Snapshot()
		writeJSON(w, http.StatusOK, snapshot)
	})

	mux.HandleFunc("/api/login", authHandler.Login)
	mux.HandleFunc("/api/logout", authHandler.Logout)
	mux.Handle("/api/me", middleware.AuthMiddleware(authService)(http.HandlerFunc(authHandler.Me)))
	mux.Handle("/api/chat", middleware.AuthMiddleware(authService)(http.HandlerFunc(chatHandler.Chat)))
	mux.Handle("/api/chat/generate-document", middleware.AuthMiddleware(authService)(http.HandlerFunc(chatHandler.GenerateDocument)))
	mux.Handle("/api/search", middleware.AuthMiddleware(authService)(http.HandlerFunc(knowledgeHandler.Search)))
	// /api/tasks dispatches by HTTP method: POST creates, GET lists
	mux.HandleFunc("/api/tasks", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost {
			middleware.AuthMiddleware(authService)(http.HandlerFunc(workflowHandler.CreateTask)).ServeHTTP(w, r)
		} else {
			middleware.AuthMiddleware(authService)(http.HandlerFunc(workflowHandler.ListTasks)).ServeHTTP(w, r)
		}
	})
	mux.Handle("/api/tasks/{id}", middleware.AuthMiddleware(authService)(http.HandlerFunc(workflowHandler.GetTask)))
	mux.Handle("/api/tasks/{id}/process", middleware.AuthMiddleware(authService)(http.HandlerFunc(workflowHandler.ProcessTask)))
	mux.Handle("/api/tasks/{id}/message", middleware.AuthMiddleware(authService)(http.HandlerFunc(workflowHandler.ProcessMessage)))
	mux.Handle("/api/tasks/{id}/data", middleware.AuthMiddleware(authService)(http.HandlerFunc(workflowHandler.SetData)))
	mux.Handle("/api/tasks/{id}/validate", middleware.AuthMiddleware(authService)(http.HandlerFunc(workflowHandler.ValidateTask)))
	mux.Handle("/api/tasks/{id}/generate", middleware.AuthMiddleware(authService)(http.HandlerFunc(workflowHandler.GenerateDocument)))
	mux.Handle("/api/tasks/{id}/sources", middleware.AuthMiddleware(authService)(http.HandlerFunc(workflowHandler.GetTaskSources)))
	mux.Handle("/api/documents", middleware.AuthMiddleware(authService)(http.HandlerFunc(workflowHandler.ListDocuments)))
	mux.Handle("/api/documents/{id}/sources", middleware.AuthMiddleware(authService)(http.HandlerFunc(workflowHandler.GetDocumentSources)))
	mux.Handle("/api/documents/{id}/docx", middleware.AuthMiddleware(authService)(http.HandlerFunc(workflowHandler.DownloadDocx)))
	mux.Handle("/api/documents/{id}/pdf", middleware.AuthMiddleware(authService)(http.HandlerFunc(workflowHandler.DownloadPdf)))
	mux.Handle("/api/history", middleware.AuthMiddleware(authService)(http.HandlerFunc(workflowHandler.ListHistory)))
	fmt.Println("[MAIN] Rotas registradas")

	addr := ":" + cfg.Port
	fmt.Printf("[MAIN] Iniciando servidor HTTP em %s\n", addr)
	slog.Info("server started", "addr", addr)
	if err := http.ListenAndServe(addr,
		corsMiddleware(cfg.FrontendURLs)(
			metrics.RequestIDMiddleware(
				metrics.LoggingMiddleware(
					middleware.WithJWTSecret(cfg.JWTSecret)(mux),
				),
			),
		),
	); err != nil {
		fmt.Printf("[MAIN] Erro no servidor HTTP: %v\n", err)
		slog.Error("server stopped", "error", err)
	}
}

func corsMiddleware(allowedOrigins []string) func(http.Handler) http.Handler {
	allowed := make(map[string]bool)
	for _, o := range allowedOrigins {
		allowed[o] = true
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			origin := r.Header.Get("Origin")
			slog.Debug("CORS request", "method", r.Method, "path", r.URL.Path, "origin", origin)

			// Check if origin is allowed
			allowedOrigin := ""
			if origin != "" && allowed[origin] {
				allowedOrigin = origin
			}

			if allowedOrigin == "" {
				// No allowed origin, don't set CORS headers (will block cross-origin requests)
				slog.Warn("CORS origin not allowed", "origin", origin, "path", r.URL.Path)
			} else {
				w.Header().Set("Access-Control-Allow-Origin", allowedOrigin)
				w.Header().Set("Access-Control-Allow-Methods", "POST, GET, OPTIONS")
				w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
				w.Header().Set("Access-Control-Allow-Credentials", "true")
			}

			if r.Method == "OPTIONS" {
				slog.Debug("CORS preflight OPTIONS")
				w.WriteHeader(http.StatusOK)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

func loggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		fmt.Printf("[LOG] Inicio request: %s %s\n", r.Method, r.URL.Path)
		next.ServeHTTP(w, r)
		slog.Info("request",
			"method", r.Method,
			"path", r.URL.Path,
			"duration", time.Since(start),
		)
		fmt.Printf("[LOG] Fim request: %s %s duracao=%v\n", r.Method, r.URL.Path, time.Since(start))
	})
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}
