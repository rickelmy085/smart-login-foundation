package web

import "context"

// Client é um cliente simples para busca web via scraping.
type Client struct{}

func NewClient() *Client {
	return &Client{}
}

func (c *Client) Search(ctx context.Context, query string) ([]SearchResult, error) {
	return Search(ctx, query)
}
