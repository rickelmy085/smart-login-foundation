// Package repository é a camada de acesso ao banco (queries SQL).
// Aqui só se executa SQL; nenhuma regra de negócio mora aqui.
package repository

import (
	"context"
	"database/sql"

	"github.com/bsmart/abis/internal/models"
)

// EmployeeRepo faz queries na tabela "employees".
// Guarda o *sql.DB para reuso (conexões são gerenciadas internamente).
type EmployeeRepo struct {
	db *sql.DB
}

// Construtor idiomático em Go: New + tipo.
// Retornar ponteiro (*EmployeeRepo) evita copiar o struct a cada chamada.
func NewEmployeeRepo(db *sql.DB) *EmployeeRepo {
	return &EmployeeRepo{db: db}
}

// FindByRE busca um employee ativo pelo RE.
// O "?" é placeholder parametrizado: protege contra SQL injection.
func (r *EmployeeRepo) FindByRE(ctx context.Context, re string) (models.Employee, error) {
	var e models.Employee
	err := r.db.QueryRowContext(ctx, `SELECT * FROM employees WHERE re = ? AND active = 1`, re).Scan(
		&e.ID, &e.RE, &e.Name, &e.Role, &e.Email, &e.Password, &e.Active, &e.CreatedAt, &e.UpdatedAt,
	)
	if err != nil {
		return models.Employee{}, err
	}
	return e, nil
}

// FindByID busca um employee ativo pelo ID interno.
func (r *EmployeeRepo) FindByID(ctx context.Context, id string) (models.Employee, error) {
	var e models.Employee
	err := r.db.QueryRowContext(ctx, `SELECT * FROM employees WHERE id = ? AND active = 1`, id).Scan(
		&e.ID, &e.RE, &e.Name, &e.Role, &e.Email, &e.Password, &e.Active, &e.CreatedAt, &e.UpdatedAt,
	)
	if err != nil {
		return models.Employee{}, err
	}
	return e, nil
}

// SessionRepo faz queries na tabela "sessions".
type SessionRepo struct {
	db *sql.DB
}

func NewSessionRepo(db *sql.DB) *SessionRepo {
	return &SessionRepo{db: db}
}

// Create insere uma nova sessão.
func (r *SessionRepo) Create(ctx context.Context, s models.Session) error {
	_, err := r.db.ExecContext(ctx, `
		INSERT INTO sessions (id, employee_id, expires_at)
		VALUES (?, ?, ?)
	`, s.ID, s.EmployeeID, s.ExpiresAt)
	return err
}

// FindByID busca uma sessão pelo ID (UUID).
func (r *SessionRepo) FindByID(ctx context.Context, id string) (models.Session, error) {
	var s models.Session
	err := r.db.QueryRowContext(ctx, `SELECT * FROM sessions WHERE id = ?`, id).Scan(
		&s.ID, &s.EmployeeID, &s.ExpiresAt, &s.CreatedAt,
	)
	if err != nil {
		return models.Session{}, err
	}
	return s, nil
}

// Delete remove uma sessão específica (logout).
func (r *SessionRepo) Delete(ctx context.Context, id string) error {
	_, err := r.db.ExecContext(ctx, `DELETE FROM sessions WHERE id = ?`, id)
	return err
}

// DeleteByEmployeeID remove todas as sessões de um employee
// (ex: "sair de todos os dispositivos").
func (r *SessionRepo) DeleteByEmployeeID(ctx context.Context, employeeID string) error {
	_, err := r.db.ExecContext(ctx, `DELETE FROM sessions WHERE employee_id = ?`, employeeID)
	return err
}
