// Package models define as structs de dados.
// Em Go, structs aqui têm papel duplo:
//   1) Representar linhas do banco (com tags `db:"..."`).
//   2) Representar o JSON da API (com tags `json:"..."`).
package models

// Employee é um colaborador do banco (linha da tabela "employees").
type Employee struct {
	ID       string `db:"id" json:"id"`              // chave primária
	RE       string `db:"re" json:"re"`              // Registro de Empregado (login)
	Name     string `db:"name" json:"name"`          // nome completo
	Role     string `db:"role" json:"role"`          // cargo / papel
	Email    string `db:"email" json:"email"`        // e-mail corporativo
	Password string `db:"password" json:"-"`         // senha (NUNCA vai no JSON; "-" esconde)
	Active   bool   `db:"active" json:"active"`      // 0/1 no banco, bool no Go
	CreatedAt string `db:"created_at" json:"created_at"`
	UpdatedAt string `db:"updated_at" json:"updated_at"`
}

// Session é uma sessão ativa de login (linha da tabela "sessions").
// Cada login bem-sucedido cria uma linha; o token JWT carrega o ID dela.
type Session struct {
	ID         string `db:"id" json:"id"`                    // UUID da sessão
	EmployeeID string `db:"employee_id" json:"employee_id"`  // FK -> employees.id
	ExpiresAt  string `db:"expires_at" json:"expires_at"`    // string (formato SQLite)
	CreatedAt  string `db:"created_at" json:"created_at"`
}

// LoginRequest é o corpo JSON esperado em POST /api/login.
type LoginRequest struct {
	RE       string `json:"re"`       // RE digitado
	Password string `json:"password"` // senha digitada
	Remember bool   `json:"remember"` // se true, sessão mais longa (7 dias)
}

// LoginResponse é o JSON devolvido em POST /api/login bem-sucedido.
// Em Go, "embutir" um struct (sem dar nome) traz seus campos para o JSON:
//   Token + Session (id, employee_id, expires_at, created_at) + Employee.
type LoginResponse struct {
	Token    string   `json:"token"`
	Session            // campos inline (id, employee_id, expires_at, created_at)
	Employee Employee `json:"employee"`
}

// MeResponse é o JSON devolvido em GET /api/me.
// "Embutir Employee" (sem nome) promove os campos dela pro topo.
type MeResponse struct {
	Employee
}
