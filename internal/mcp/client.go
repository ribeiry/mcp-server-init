package mcp

import "context"

type Tool struct {
	Name        string `json:"name"`
	Description string `json:"description"`
}

type Client interface {
	ListTools(ctx context.Context) ([]Tool, error)
	ReadFile(ctx context.Context, path string) (string, error)
	Close() error
}
