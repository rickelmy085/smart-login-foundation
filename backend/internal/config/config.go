// Package config carrega configurações do ambiente (env vars) e
// expõe uma struct com os valores usados pelo resto do app.
package config

import (
	"fmt"
	"os" // usado para ler variáveis de ambiente
	"strconv"
	"strings"
)

// Config agrupa todas as configurações em uma única struct.
type Config struct {
	Port         string   // porta do servidor HTTP (ex: "8081")
	DatabasePath string   // caminho do arquivo SQLite (ex: "./data/abis.db")
	JWTSecret    string   // chave secreta usada para assinar tokens JWT
	FrontendURLs []string // origens permitidas no CORS (ex: ["http://localhost:8080", "https://app.exemplo.com"])
	GroqAPIKey   string   // chave da API Groq
	GroqModel    string   // modelo Groq (ex: "openai/gpt-oss-120b")
	Debug        bool     // modo debug

	DocumentEngineEnabled     bool   // habilita o Document Engine Python
	DocumentEngineURL         string // URL do Document Engine (ex: "http://127.0.0.1:8090")
	DocumentEngineSecret      string // segredo interno para autenticação
	DocumentEngineTimeout     int    // timeout em segundos
	DocumentEngineFallback    bool   // fallback para gerador Go quando Python falha
}

// Load lê as variáveis de ambiente e devolve uma Config preenchida.
func Load() Config {
	cfg := Config{
		Port:         env("PORT", "8081"),
		DatabasePath: env("DATABASE_PATH", "./data/abis.db"),
		JWTSecret:    env("JWT_SECRET", "change-me-in-env"),
		FrontendURLs: parseOrigins(env("FRONTEND_URLS", "http://localhost:8080")),
		GroqAPIKey:   env("GROQ_API_KEY", ""),
		GroqModel:    env("GROQ_MODEL", "openai/gpt-oss-120b"),
		Debug:        env("DEBUG", "false") == "true",

		DocumentEngineEnabled:  env("DOCUMENT_ENGINE_ENABLED", "false") == "true",
		DocumentEngineURL:      env("DOCUMENT_ENGINE_URL", "http://127.0.0.1:8090"),
		DocumentEngineSecret:   env("DOCUMENT_ENGINE_INTERNAL_SECRET", "change-me"),
		DocumentEngineTimeout:  envInt("DOCUMENT_ENGINE_TIMEOUT_SECONDS", 30),
		DocumentEngineFallback: env("DOCUMENT_ENGINE_FALLBACK_TO_GO", "true") == "true",
	}
	fmt.Printf("[CONFIG] Load: porta=%s db=%s jwt_secret_length=%d frontend_urls=%v groq_model=%s debug=%v\n",
		cfg.Port, cfg.DatabasePath, len(cfg.JWTSecret), cfg.FrontendURLs, cfg.GroqModel, cfg.Debug)
	if cfg.JWTSecret == "" {
		panic("JWT_SECRET must not be empty")
	}
	return cfg
}

// env é um helper: lê a variável de ambiente `key`.
// Se não estiver definida ou for string vazia, retorna `fallback`.
func env(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		fmt.Printf("[CONFIG] env %s encontrada\n", key)
		return v
	}
	fmt.Printf("[CONFIG] env %s nao encontrada, usando fallback=%s\n", key, fallback)
	return fallback
}

// envInt lê uma variável de ambiente como inteiro.
// Se não estiver definida ou não for parseável, retorna fallback.
func envInt(key string, fallback int) int {
	v := os.Getenv(key)
	if v == "" {
		return fallback
	}
	n, err := strconv.Atoi(v)
	if err != nil {
		fmt.Printf("[CONFIG] env %s com valor invalido %q, usando fallback=%d\n", key, v, fallback)
		return fallback
	}
	return n
}

// parseOrigins parses a comma-separated list of origins.
func parseOrigins(s string) []string {
	if s == "" {
		return []string{}
	}
	var result []string
	for _, part := range strings.Split(s, ",") {
		trimmed := strings.TrimSpace(part)
		if trimmed != "" {
			result = append(result, trimmed)
		}
	}
	return result
}
