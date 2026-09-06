package repository

import (
	"context"
	"database/sql"
	"fmt"
)

// ChatMessage representa uma mensagem de chat persistida.
type ChatMessage struct {
	ID             string
	EmployeeID     string
	Role           string
	Content        string
	AllowWebSearch bool
	SourcesCount   int
	CreatedAt      string
}

// ChatRepo gerencia mensagens de chat no SQLite.
type ChatRepo struct {
	db *sql.DB
}

// NewChatRepo cria um novo repositório de chat.
func NewChatRepo(db *sql.DB) *ChatRepo {
	return &ChatRepo{db: db}
}

// InsertMessage persiste uma mensagem de chat.
func (r *ChatRepo) InsertMessage(ctx context.Context, m ChatMessage) error {
	_, err := r.db.ExecContext(ctx, `
		INSERT INTO chat_messages (id, employee_id, role, content, allow_web_search, sources_count, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?)
	`, m.ID, m.EmployeeID, m.Role, m.Content, boolToInt(m.AllowWebSearch), m.SourcesCount, m.CreatedAt)
	if err != nil {
		return fmt.Errorf("insert chat message: %w", err)
	}
	return nil
}

// ListByEmployee retorna as últimas mensagens de um funcionário.
func (r *ChatRepo) ListByEmployee(ctx context.Context, employeeID string, limit int) ([]ChatMessage, error) {
	if limit <= 0 {
		limit = 50
	}
	rows, err := r.db.QueryContext(ctx, `
		SELECT id, employee_id, role, content, allow_web_search, sources_count, created_at
		FROM chat_messages
		WHERE employee_id = ?
		ORDER BY created_at DESC
		LIMIT ?
	`, employeeID, limit)
	if err != nil {
		return nil, fmt.Errorf("list chat messages: %w", err)
	}
	defer rows.Close()

	var result []ChatMessage
	for rows.Next() {
		var m ChatMessage
		var allowWebSearchInt int
		if err := rows.Scan(&m.ID, &m.EmployeeID, &m.Role, &m.Content, &allowWebSearchInt, &m.SourcesCount, &m.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan chat message: %w", err)
		}
		m.AllowWebSearch = allowWebSearchInt != 0
		result = append(result, m)
	}
	return result, rows.Err()
}

func boolToInt(v bool) int {
	if v {
		return 1
	}
	return 0
}
