// Package groq é um cliente HTTP minimalista para a API da Groq
// (compatível com OpenAI Chat Completions).
//
// Por que não usamos uma lib pronta?
//   - Para uma única chamada (chat completions) o stdlib net/http é suficiente.
//   - Evita dependências externas e binário maior.
//   - Mantém o código auditável: você vê exatamente o que sai/entra.
package groq

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"time"
)

const (
	// Endpoint oficial da Groq (compatível com OpenAI).
	DefaultEndpoint = "https://api.groq.com/openai/v1/chat/completions"

	// Modelo padrão. Pode ser sobrescrito via config.
	DefaultModel = "openai/gpt-oss-120b"

	// Limites defensivos para não estourar a janela de contexto
	// do modelo ou a cota da API.
	DefaultMaxTokens   = 1024
	DefaultTemperature = 0.2 // baixa temperatura = respostas mais determinísticas
)

// Message representa uma mensagem no formato Chat Completions da OpenAI/Groq.
// Roles aceitos: "system", "user", "assistant".
type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// ChatRequest é o corpo enviado para /chat/completions.
type ChatRequest struct {
	Model       string    `json:"model"`
	Messages    []Message `json:"messages"`
	MaxTokens   int       `json:"max_tokens,omitempty"`
	Temperature float64   `json:"temperature,omitempty"`
	Stream      bool      `json:"stream,omitempty"` // sempre false por enquanto
}

// Choice é uma escolha devolvida pelo modelo.
type Choice struct {
	Index        int     `json:"index"`
	Message      Message `json:"message"`
	FinishReason string  `json:"finish_reason"`
}

// ChatResponse é o JSON devolvido pela Groq em sucesso.
type ChatResponse struct {
	ID      string   `json:"id"`
	Model   string   `json:"model"`
	Choices []Choice `json:"choices"`
	Usage   struct {
		PromptTokens     int `json:"prompt_tokens"`
		CompletionTokens int `json:"completion_tokens"`
		TotalTokens      int `json:"total_tokens"`
	} `json:"usage"`
}

// Client é o cliente Groq. Reutilizável entre requests (http.Client com pool).
type Client struct {
	apiKey     string
	endpoint   string
	model      string
	maxTokens  int
	temperature float64
	httpClient *http.Client
}

// New monta um Client. apiKey é obrigatório (em produção vem do .env).
func New(apiKey, model string) *Client {
	if model == "" {
		model = DefaultModel
	}
	return &Client{
		apiKey:      apiKey,
		endpoint:    DefaultEndpoint,
		model:       model,
		maxTokens:   DefaultMaxTokens,
		temperature: DefaultTemperature,
		httpClient: &http.Client{
			Timeout: 60 * time.Second,
		},
	}
}

// Empty returns true if the client has no API key configured.
func (c *Client) Empty() bool {
	return c == nil || strings.TrimSpace(c.apiKey) == ""
}

// WithMaxTokens e WithTemperature permitem ajustes finos sem mexer no construtor.
func (c *Client) WithMaxTokens(n int) *Client {
	if n > 0 {
		c.maxTokens = n
	}
	return c
}

func (c *Client) WithTemperature(t float64) *Client {
	if t >= 0 && t <= 2 {
		c.temperature = t
	}
	return c
}

// ErrEmptyAPIKey é retornado quando o cliente foi criado sem chave.
type ErrEmptyAPIKey struct{}

func (ErrEmptyAPIKey) Error() string { return "groq: api key is empty" }

// Chat envia uma conversa para a Groq e devolve o texto da primeira escolha.
// Retorna erro amigável se a chave estiver ausente ou a API responder algo inesperado.
func (c *Client) Chat(ctx context.Context, messages []Message) (string, error) {
	return c.ChatWithMaxTokens(ctx, messages, c.maxTokens)
}

// ChatWithMaxTokens allows overriding the max tokens for a single call.
func (c *Client) ChatWithMaxTokens(ctx context.Context, messages []Message, maxTokens int) (string, error) {
	fmt.Printf("[GROQ] ChatWithMaxTokens iniciado model=%s maxTokens=%d messages=%d\n", c.model, maxTokens, len(messages))
	if strings.TrimSpace(c.apiKey) == "" {
		fmt.Println("[GROQ] ChatWithMaxTokens apiKey vazia")
		return "", ErrEmptyAPIKey{}
	}
	if len(messages) == 0 {
		fmt.Println("[GROQ] ChatWithMaxTokens messages vazia")
		return "", fmt.Errorf("groq: messages is empty")
	}

	if maxTokens <= 0 {
		maxTokens = c.maxTokens
	}

	req := ChatRequest{
		Model:       c.model,
		Messages:    messages,
		MaxTokens:   maxTokens,
		Temperature: c.temperature,
	}
	body, err := json.Marshal(req)
	if err != nil {
		fmt.Printf("[GROQ] ChatWithMaxTokens erro marshal: %v\n", err)
		return "", fmt.Errorf("marshal request: %w", err)
	}
	fmt.Printf("[GROQ] ChatWithMaxTokens request body length=%d\n", len(body))

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, c.endpoint, bytes.NewReader(body))
	if err != nil {
		fmt.Printf("[GROQ] ChatWithMaxTokens erro new request: %v\n", err)
		return "", fmt.Errorf("new request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+c.apiKey)

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		fmt.Printf("[GROQ] ChatWithMaxTokens erro http call: %v\n", err)
		return "", fmt.Errorf("groq http call: %w", err)
	}
	defer resp.Body.Close()
	fmt.Printf("[GROQ] ChatWithMaxTokens response status=%d\n", resp.StatusCode)

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		fmt.Printf("[GROQ] ChatWithMaxTokens erro read body: %v\n", err)
		return "", fmt.Errorf("read body: %w", err)
	}
	fmt.Printf("[GROQ] ChatWithMaxTokens response body length=%d\n", len(respBody))

	if resp.StatusCode != http.StatusOK {
		// Não expomos o body inteiro ao usuário — logamos e devolvemos msg curta.
		slog.Warn("groq non-200", "status", resp.StatusCode, "body", string(respBody))
		fmt.Printf("[GROQ] ChatWithMaxTokens status nao-200: %d body=%s\n", resp.StatusCode, string(respBody))
		return "", fmt.Errorf("groq returned status %d", resp.StatusCode)
	}

	var parsed ChatResponse
	if err := json.Unmarshal(respBody, &parsed); err != nil {
		fmt.Printf("[GROQ] ChatWithMaxTokens erro unmarshal: %v\n", err)
		return "", fmt.Errorf("unmarshal response: %w", err)
	}
	if len(parsed.Choices) == 0 {
		fmt.Println("[GROQ] ChatWithMaxTokens nenhuma choice na resposta")
		return "", fmt.Errorf("groq returned no choices")
	}
	fmt.Printf("[GROQ] ChatWithMaxTokens sucesso model=%s total_tokens=%d\n", parsed.Model, parsed.Usage.TotalTokens)

	// Log de observabilidade (consumo de tokens) — útil para debugar custos.
	slog.Info("groq chat",
		"model", parsed.Model,
		"prompt_tokens", parsed.Usage.PromptTokens,
		"completion_tokens", parsed.Usage.CompletionTokens,
		"total_tokens", parsed.Usage.TotalTokens,
	)

	return parsed.Choices[0].Message.Content, nil
}
