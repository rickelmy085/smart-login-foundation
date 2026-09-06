// Package repository é a camada de acesso ao banco (queries SQL).
// Aqui só se executa SQL; nenhuma regra de negócio mora aqui.
package repository

import (
	"context"
	"database/sql"
	"fmt" // debug prints

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
	fmt.Printf("[REPO] FindByRE re=%s\n", re)
	var e models.Employee
	err := r.db.QueryRowContext(ctx, `SELECT * FROM employees WHERE re = ? AND active = 1`, re).Scan(
		&e.ID, &e.RE, &e.Name, &e.Role, &e.Email, &e.Password, &e.Active, &e.CreatedAt, &e.UpdatedAt,
	)
	if err != nil {
		fmt.Printf("[REPO] FindByRE erro: %v\n", err)
		return models.Employee{}, err
	}
	fmt.Printf("[REPO] FindByRE sucesso employeeID=%s name=%s\n", e.ID, e.Name)
	return e, nil
}

// FindByID busca um employee ativo pelo ID interno.
func (r *EmployeeRepo) FindByID(ctx context.Context, id string) (models.Employee, error) {
	fmt.Printf("[REPO] FindByID id=%s\n", id)
	var e models.Employee
	err := r.db.QueryRowContext(ctx, `SELECT * FROM employees WHERE id = ? AND active = 1`, id).Scan(
		&e.ID, &e.RE, &e.Name, &e.Role, &e.Email, &e.Password, &e.Active, &e.CreatedAt, &e.UpdatedAt,
	)
	if err != nil {
		fmt.Printf("[REPO] FindByID erro: %v\n", err)
		return models.Employee{}, err
	}
	fmt.Printf("[REPO] FindByID sucesso employeeID=%s name=%s\n", e.ID, e.Name)
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
	fmt.Printf("[REPO] Session Create sessionID=%s employeeID=%s expiresAt=%s\n", s.ID, s.EmployeeID, s.ExpiresAt)
	_, err := r.db.ExecContext(ctx, `
		INSERT INTO sessions (id, employee_id, expires_at)
		VALUES (?, ?, ?)
	`, s.ID, s.EmployeeID, s.ExpiresAt)
	if err != nil {
		fmt.Printf("[REPO] Session Create erro: %v\n", err)
		return err
	}
	fmt.Printf("[REPO] Session Create sucesso sessionID=%s\n", s.ID)
	return nil
}

// FindByID busca uma sessão pelo ID (UUID).
func (r *SessionRepo) FindByID(ctx context.Context, id string) (models.Session, error) {
	fmt.Printf("[REPO] Session FindByID sessionID=%s\n", id)
	var s models.Session
	err := r.db.QueryRowContext(ctx, `SELECT * FROM sessions WHERE id = ?`, id).Scan(
		&s.ID, &s.EmployeeID, &s.ExpiresAt, &s.CreatedAt,
	)
	if err != nil {
		fmt.Printf("[REPO] Session FindByID erro: %v\n", err)
		return models.Session{}, err
	}
	fmt.Printf("[REPO] Session FindByID sucesso sessionID=%s employeeID=%s\n", s.ID, s.EmployeeID)
	return s, nil
}

// Delete remove uma sessão específica (logout).
func (r *SessionRepo) Delete(ctx context.Context, id string) error {
	fmt.Printf("[REPO] Session Delete sessionID=%s\n", id)
	_, err := r.db.ExecContext(ctx, `DELETE FROM sessions WHERE id = ?`, id)
	if err != nil {
		fmt.Printf("[REPO] Session Delete erro: %v\n", err)
		return err
	}
	fmt.Printf("[REPO] Session Delete sucesso sessionID=%s\n", id)
	return nil
}

// DeleteByEmployeeID remove todas as sessões de um employee
// (ex: "sair de todos os dispositivos").
func (r *SessionRepo) DeleteByEmployeeID(ctx context.Context, employeeID string) error {
	fmt.Printf("[REPO] Session DeleteByEmployeeID employeeID=%s\n", employeeID)
	_, err := r.db.ExecContext(ctx, `DELETE FROM sessions WHERE employee_id = ?`, employeeID)
	if err != nil {
		fmt.Printf("[REPO] Session DeleteByEmployeeID erro: %v\n", err)
		return err
	}
	fmt.Printf("[REPO] Session DeleteByEmployeeID sucesso employeeID=%s\n", employeeID)
	return nil
}
