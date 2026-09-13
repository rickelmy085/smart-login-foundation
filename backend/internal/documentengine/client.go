package documentengine

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

	"github.com/bsmart/abis/internal/docspec"
)

type Client struct {
	baseURL    string
	secret     string
	httpClient *http.Client
}

func NewClient(baseURL, secret string, timeout time.Duration) *Client {
	if timeout <= 0 {
		timeout = 30 * time.Second
	}
	return &Client{
		baseURL: strings.TrimRight(baseURL, "/"),
		secret:  secret,
		httpClient: &http.Client{
			Timeout: timeout,
		},
	}
}

func (c *Client) Health(ctx context.Context) (*HealthResponse, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL+"/health", nil)
	if err != nil {
		return nil, fmt.Errorf("create health request: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("health request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("health returned status %d", resp.StatusCode)
	}

	var result HealthResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("decode health response: %w", err)
	}
	return &result, nil
}

func (c *Client) Generate(ctx context.Context, spec *docspec.DocumentSpec) (*GenerateResponse, error) {
	if err := docspec.Validate(spec); err != nil {
		return nil, fmt.Errorf("invalid spec: %w", err)
	}

	body, err := json.Marshal(spec)
	if err != nil {
		return nil, fmt.Errorf("marshal spec: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/generate", bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("create generate request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	if c.secret != "" {
		req.Header.Set("X-Internal-Secret", c.secret)
	}

	start := time.Now()
	resp, err := c.httpClient.Do(req)
	elapsed := time.Since(start)

	if err != nil {
		slog.Error("document engine request failed",
			"run_id", spec.RunID,
			"spec_id", spec.SpecID,
			"error", err.Error(),
			"elapsed", elapsed,
		)
		return nil, fmt.Errorf("generate request failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(io.LimitReader(resp.Body, 100*1024*1024))
	if err != nil {
		return nil, fmt.Errorf("read response body: %w", err)
	}

	slog.Info("document engine response",
		"run_id", spec.RunID,
		"spec_id", spec.SpecID,
		"status", resp.StatusCode,
		"elapsed", elapsed,
		"body_size", len(respBody),
	)

	if resp.StatusCode != http.StatusOK {
		var errDetail ErrorDetail
		if json.Unmarshal(respBody, &errDetail) == nil && errDetail.ErrorCode != "" {
			return nil, &EngineError{
				HTTPStatus: resp.StatusCode,
				Code:       errDetail.ErrorCode,
				Message:    errDetail.Message,
			}
		}
		return nil, fmt.Errorf("document engine returned status %d: %s", resp.StatusCode, truncate(string(respBody), 500))
	}

	var result GenerateResponse
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, fmt.Errorf("decode generate response: %w", err)
	}

	return &result, nil
}

type EngineError struct {
	HTTPStatus int
	Code       string
	Message    string
}

func (e *EngineError) Error() string {
	return fmt.Sprintf("document engine error [%s]: %s (HTTP %d)", e.Code, e.Message, e.HTTPStatus)
}

func IsEngineError(err error) (*EngineError, bool) {
	if err == nil {
		return nil, false
	}
	if e, ok := err.(*EngineError); ok {
		return e, true
	}
	return nil, false
}

func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}
