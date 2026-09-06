// Package config carrega configurações do ambiente (env vars) e
// expõe uma struct com os valores usados pelo resto do app.
package config

import (
	"fmt"
	"os" // usado para ler variáveis de ambiente
)

// Config agrupa todas as configurações em uma única struct.
type Config struct {
	Port         string // porta do servidor HTTP (ex: "8081")
	DatabasePath string // caminho do arquivo SQLite (ex: "./data/abis.db")
	JWTSecret    string // chave secreta usada para assinar tokens JWT
	FrontendURL  string // origem permitida no CORS (ex: "http://localhost:8080")
	GroqAPIKey   string // chave da API Groq
	GroqModel    string // modelo Groq (ex: "openai/gpt-oss-120b")
	Debug        bool   // modo debug
}

// Load lê as variáveis de ambiente e devolve uma Config preenchida.
func Load() Config {
	cfg := Config{
		Port:         env("PORT", "8081"),
		DatabasePath: env("DATABASE_PATH", "./data/abis.db"),
		JWTSecret:    env("JWT_SECRET", "change-me-in-env"),
		FrontendURL:  env("FRONTEND_URL", "http://localhost:8080"),
		GroqAPIKey:   env("GROQ_API_KEY", ""),
		GroqModel:    env("GROQ_MODEL", "openai/gpt-oss-120b"),
		Debug:        env("DEBUG", "false") == "true",
	}
	fmt.Printf("[CONFIG] Load: porta=%s db=%s jwt_secret_length=%d frontend=%s groq_model=%s debug=%v\n",
		cfg.Port, cfg.DatabasePath, len(cfg.JWTSecret), cfg.FrontendURL, cfg.GroqModel, cfg.Debug)
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
