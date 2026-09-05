// Continuação do package database: criação de tabelas e dados iniciais.
package database

import (
	"context" // permite cancelar/cumprir deadlines nas queries
	"database/sql"
	"log/slog"
)

// Migrate cria as tabelas necessárias (se não existirem) e popula dados
// de exemplo. Idempotente: rodar várias vezes não dá erro.
//
// Fluxo:
//   1) Abre uma transação (tudo ou nada).
//   2) Cria tabela employees.
//   3) Cria tabela sessions.
//   4) Cria índice em sessions.employee_id (acelera lookups).
//   5) Insere o usuário "demo" se a tabela estiver vazia.
//   6) Faz commit.
//
// Se qualquer passo falhar, o defer Rollback() desfaz tudo.
func Migrate(ctx context.Context, db *sql.DB) error {
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	// Rollback é no-op se o commit já foi feito, então é seguro sempre chamar.
	defer func() { _ = tx.Rollback() }()

	// Tabela de colaboradores. Observações:
	//   - id é TEXT (UUID/legível, não autoincrement).
	//   - re é o "Registro de Empregado" (login).
	//   - password armazena a senha em texto puro (decisão de MVP).
	//   - active permite "desligar" um usuário sem deletá-lo.
	if _, err := tx.ExecContext(ctx, `
		CREATE TABLE IF NOT EXISTS employees (
			id TEXT PRIMARY KEY,
			re TEXT NOT NULL UNIQUE,
			name TEXT NOT NULL,
			role TEXT NOT NULL,
			email TEXT NOT NULL UNIQUE,
			password TEXT NOT NULL,
			active INTEGER NOT NULL DEFAULT 1,
			created_at TEXT NOT NULL DEFAULT (datetime('now')),
			updated_at TEXT NOT NULL DEFAULT (datetime('now'))
		);
	`); err != nil {
		return err
	}

	// Tabela de sessões. Cada login bem-sucedido cria uma linha aqui.
	// O token JWT carrega o session_id; se a sessão for deletada/expirada
	// no banco, o token deixa de valer, mesmo que a assinatura bata.
	if _, err := tx.ExecContext(ctx, `
		CREATE TABLE IF NOT EXISTS sessions (
			id TEXT PRIMARY KEY,
			employee_id TEXT NOT NULL REFERENCES employees(id),
			expires_at TEXT NOT NULL,
			created_at TEXT NOT NULL DEFAULT (datetime('now'))
		);
	`); err != nil {
		return err
	}

	// Índice acelera queries do tipo "todas as sessões de um employee".
	if _, err := tx.ExecContext(ctx, `CREATE INDEX IF NOT EXISTS idx_sessions_employee_id ON sessions(employee_id);`); err != nil {
		return err
	}

	// Seed: garante um usuário de demonstração.
	if err := seed(ctx, tx); err != nil {
		return err
	}

	slog.Info("seed finished")

	return tx.Commit()
}

// seed insere o usuário demo apenas se a tabela estiver vazia.
// Como as migrations rodam a cada startup, isso evita duplicar.
func seed(ctx context.Context, tx *sql.Tx) error {
	var count int
	if err := tx.QueryRowContext(ctx, `SELECT count(*) FROM employees`).Scan(&count); err != nil {
		return err
	}
	slog.Info("seed check", "count", count)
	if count > 0 {
		return nil // já tem usuário, nada a fazer
	}

	slog.Info("seed inserting demo employee", "re", "123456")

	// Insere o "Colaborador Demo" com RE 123456 e senha demo123.
	// Senha em texto puro por decisão de MVP (sem bcrypt).
	_, err := tx.ExecContext(ctx, `
		INSERT INTO employees (id, re, name, role, email, password)
		VALUES (?, ?, ?, ?, ?, ?)
	`, "emp-001", "123456", "Colaborador Demo", "Analista", "re123456@bradesco.com.br", "demo123")
	if err != nil {
		return err
	}

	return nil
}
