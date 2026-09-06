package web

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"time"
)

const (
	userAgent = "Mozilla/5.0 (compatible; ABIS/1.0; +https://example.com)"
	timeout   = 10 * time.Second
)

var (
	snippetRe = regexp.MustCompile(`<[^>]+>`)
)

// SearchResult representa um resultado de busca web.
type SearchResult struct {
	Title   string
	URL     string
	Snippet string
}

// Search executa uma busca simples usando DuckDuckGo HTML.
func Search(ctx context.Context, query string) ([]SearchResult, error) {
	if strings.TrimSpace(query) == "" {
		return nil, nil
	}

	searchURL := "https://html.duckduckgo.com/html/?q=" + url.QueryEscape(query)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, searchURL, nil)
	if err != nil {
		return nil, fmt.Errorf("web search request: %w", err)
	}
	req.Header.Set("User-Agent", userAgent)
	req.Header.Set("Accept", "text/html,application/xhtml+xml")
	req.Header.Set("Accept-Language", "pt-BR,pt;q=0.9,en;q=0.8")

	client := &http.Client{Timeout: timeout}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("web search fetch: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("web search status: %s", resp.Status)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("web search read: %w", err)
	}

	html := string(body)
	return parseResults(html), nil
}

func parseResults(html string) []SearchResult {
	var results []SearchResult

	// Padrão simplificado para extrair resultados do DuckDuckGo HTML.
	re := regexp.MustCompile(`<a[^>]+class="result__a"[^>]*href="([^"]+)"[^>]*>(.*?)</a>`)
	matches := re.FindAllStringSubmatch(html, -1)

	for _, m := range matches {
		if len(m) < 3 {
			continue
		}
		link := m[1]
		title := snippetRe.ReplaceAllString(m[2], "")
		title = strings.TrimSpace(title)
		if title == "" {
			continue
		}

		// Extrai snippet associado.
		snippet := ""
		idx := strings.Index(html, m[0])
		if idx >= 0 {
			rest := html[idx:]
			snipRe := regexp.MustCompile(`<a[^>]+class="result__snippet"[^>]*>(.*?)</a>`)
			snipMatch := snipRe.FindStringSubmatch(rest)
			if len(snipMatch) >= 2 {
				snippet = snippetRe.ReplaceAllString(snipMatch[1], "")
				snippet = strings.TrimSpace(snippet)
			}
		}

		results = append(results, SearchResult{
			Title:   title,
			URL:     link,
			Snippet: snippet,
		})
	}

	return results
}

// FormatResults converte os resultados em texto para contexto do LLM.
func FormatResults(results []SearchResult) string {
	if len(results) == 0 {
		return ""
	}

	var b strings.Builder
	b.WriteString("Resultados da busca web:\n")
	for i, r := range results {
		b.WriteString(fmt.Sprintf("\n[%d] %s\n%s\nURL: %s\n", i+1, r.Title, r.Snippet, r.URL))
	}
	return b.String()
}
