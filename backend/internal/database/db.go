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
	// Garante que a pasta do arquivo existe (ex: cria "./data/").
	dir := filepath.Dir(databasePath)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, fmt.Errorf("create database dir: %w", err)
	}

	// sql.Open não abre o arquivo de fato; só prepara o handle.
	// A conexão real acontece quando a primeira query roda.
	db, err := sql.Open("sqlite", databasePath)
	if err != nil {
		return nil, fmt.Errorf("connect sqlite: %w", err)
	}

	// Limita a 10 conexões abertas simultaneamente. SQLite lida mal
	// com concorrência alta em modo arquivo; 10 é folgado pra API leve.
	db.SetMaxOpenConns(10)
	slog.Info("database connected", "path", databasePath)
	return db, nil
}
