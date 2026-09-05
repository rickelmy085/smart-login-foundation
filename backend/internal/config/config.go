// Package config carrega configurações do ambiente (env vars) e
// expõe uma struct com os valores usados pelo resto do app.
package config

import (
	"os" // usado para ler variáveis de ambiente
)

// Config agrupa todas as configurações em uma única struct.
// Assim, basta passar `cfg` por aí em vez de várias strings soltas.
type Config struct {
	Port         string // porta do servidor HTTP (ex: "8081")
	DatabasePath string // caminho do arquivo SQLite (ex: "./data/abis.db")
	JWTSecret    string // chave secreta usada para assinar tokens JWT
}

// Load lê as variáveis de ambiente e devolve uma Config preenchida.
// Se a env var não existir, usa um valor padrão (fallback).
func Load() Config {
	cfg := Config{
		Port:         env("PORT", "8081"),
		DatabasePath: env("DATABASE_PATH", "./data/abis.db"),
		JWTSecret:    env("JWT_SECRET", "change-me-in-env"),
	}
	// Segurança mínima: nunca rodar com segredo vazio (produção).
	// Em dev, o fallback acima garante que tem algo.
	if cfg.JWTSecret == "" {
		panic("JWT_SECRET must not be empty")
	}
	return cfg
}

// env é um helper: lê a variável de ambiente `key`.
// Se não estiver definida ou for string vazia, retorna `fallback`.
func env(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
