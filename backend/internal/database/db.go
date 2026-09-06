// Package database concentra tudo que é "falar com o banco":
// abrir conexão e criar tabelas.
package database

import (
	"database/sql" // interface genérica do Go para bancos SQL
	"fmt"          // para formatar mensagens de erro com %w (encadeamento)
	"log/slog"     // logger
	"os"           // criar diretórios
	"path/filepath" // manipulação de caminhos de arquivo

	// O "_" (blank import) só registra o driver no pacote database/sql,
	// sem expor nenhum símbolo. O driver é "sqlite" (modernc.org/sqlite),
	// puro-Go (não precisa compilar com CGO).
	_ "modernc.org/sqlite"
)

// Connect abre (ou cria) o arquivo SQLite e devolve um *sql.DB pronto.
//
// Parâmetros:
//   databasePath: caminho do arquivo (ex: "./data/abis.db").
//
// Retorno:
//   *sql.DB: handle da conexão, usado para fazer queries.
//   error  : qualquer falha de abertura/criação.
func Connect(databasePath string) (*sql.DB, error) {
	fmt.Printf("[DB] Connect databasePath=%s\n", databasePath)
	// Garante que a pasta do arquivo existe (ex: cria "./data/").
	dir := filepath.Dir(databasePath)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		fmt.Printf("[DB] Connect erro mkdir: %v\n", err)
		return nil, fmt.Errorf("create database dir: %w", err)
	}

	// sql.Open não abre o arquivo de fato; só prepara o handle.
	// A conexão real acontece quando a primeira query roda.
	db, err := sql.Open("sqlite", databasePath)
	if err != nil {
		fmt.Printf("[DB] Connect erro sql.Open: %v\n", err)
		return nil, fmt.Errorf("connect sqlite: %w", err)
	}

	// Limita a 10 conexões abertas simultaneamente. SQLite lida mal
	// com concorrência alta em modo arquivo; 10 é folgado pra API leve.
	db.SetMaxOpenConns(10)

	for _, p := range []struct {
		name  string
		query string
	}{
		{"foreign_keys", "PRAGMA foreign_keys = ON"},
		{"journal_mode", "PRAGMA journal_mode = WAL"},
		{"busy_timeout", "PRAGMA busy_timeout = 5000"},
	} {
		if _, err := db.Exec(p.query); err != nil {
			fmt.Printf("[DB] Connect erro pragma %s: %v\n", p.name, err)
			return nil, fmt.Errorf("pragma %s: %w", p.name, err)
		}
		fmt.Printf("[DB] Connect pragma %s aplicado\n", p.name)
	}

	slog.Info("database connected", "path", databasePath)
	fmt.Printf("[DB] Connect sucesso databasePath=%s\n", databasePath)
	return db, nil
}
