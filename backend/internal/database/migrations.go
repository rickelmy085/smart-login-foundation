// Continuação do package database: criação de tabelas e dados iniciais.
package database

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"

	"golang.org/x/crypto/bcrypt"
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
	fmt.Println("[DB] Migrate iniciada")
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		fmt.Printf("[DB] Migrate erro begin tx: %v\n", err)
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
		fmt.Printf("[DB] Migrate erro create employees: %v\n", err)
		return err
	}
	fmt.Println("[DB] Migrate tabela employees criada/ok")

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
		fmt.Printf("[DB] Migrate erro create sessions: %v\n", err)
		return err
	}
	fmt.Println("[DB] Migrate tabela sessions criada/ok")

	// Índice acelera queries do tipo "todas as sessões de um employee".
	if _, err := tx.ExecContext(ctx, `CREATE INDEX IF NOT EXISTS idx_sessions_employee_id ON sessions(employee_id);`); err != nil {
		fmt.Printf("[DB] Migrate erro create index: %v\n", err)
		return err
	}
	fmt.Println("[DB] Migrate indice idx_sessions_employee_id criado/ok")

	// Seed: garante um usuário de demonstração.
	if err := seed(ctx, tx); err != nil {
		fmt.Printf("[DB] Migrate erro seed: %v\n", err)
		return err
	}

	slog.Info("seed finished")
	fmt.Println("[DB] Migrate seed finalizado")

	if err := tx.Commit(); err != nil {
		fmt.Printf("[DB] Migrate erro commit: %v\n", err)
		return err
	}
	fmt.Println("[DB] Migrate commit realizado com sucesso")
	return nil
}

// seed garante que o usuário demo exista e sempre tenha a senha bcrypt atualizada.
// É idempotente: se o usuário já existir, apenas atualiza a senha.
func seed(ctx context.Context, tx *sql.Tx) error {
	fmt.Println("[DB] seed iniciada")

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte("demo123"), bcrypt.DefaultCost)
	if err != nil {
		fmt.Printf("[DB] seed erro hash password: %v\n", err)
		return err
	}

	var existingID string
	err = tx.QueryRowContext(ctx, `SELECT id FROM employees WHERE re = ?`, "123456").Scan(&existingID)
	if err == nil {
		fmt.Printf("[DB] seed usuario demo ja existe id=%s, atualizando senha\n", existingID)
		_, err = tx.ExecContext(ctx, `UPDATE employees SET password = ?, updated_at = datetime('now') WHERE re = ?`, string(hashedPassword), "123456")
		if err != nil {
			fmt.Printf("[DB] seed erro update senha demo: %v\n", err)
			return err
		}
		fmt.Println("[DB] seed senha demo atualizada para bcrypt")
		return nil
	}
	if err != sql.ErrNoRows {
		fmt.Printf("[DB] seed erro ao verificar usuario demo: %v\n", err)
		return err
	}

	slog.Info("seed inserting demo employee", "re", "123456", "name", "José Alberto")
	fmt.Println("[DB] seed inserindo usuario demo")
	_, err = tx.ExecContext(ctx, `
		INSERT INTO employees (id, re, name, role, email, password)
		VALUES (?, ?, ?, ?, ?, ?)
	`, "emp-001", "123456", "José Alberto", "Analista", "jose.alberto@bradesco.com.br", string(hashedPassword))
	if err != nil {
		fmt.Printf("[DB] seed erro insert demo: %v\n", err)
		return err
	}
	fmt.Println("[DB] seed usuario demo inserido com sucesso")
	return nil
}
