package main

import (
	"context"
	"errors"
	"log"
	"mcp-lab-go/internal/mcp"
	"time"
)

func main() {
	log.Print("Init app")

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	client, err := mcp.NewFileSystemClient(ctx)
	if err != nil {
		log.Fatalf("failed to create client: %v", err)
	}
	defer client.Close()

	content, err := client.ReadFile(ctx, "data/test.txt")
	if err != nil {
		handleError(err)
		return
	}

	log.Printf("file content: %s", content)
}

func handleError(err error) {
	var connErr *mcp.ConnectionError
	var protoErr *mcp.ProtocolError
	var toolErr *mcp.ToolError

	switch {
	case errors.As(err, &connErr):
		log.Printf("connection error: %v", connErr)

	case errors.As(err, &protoErr):
		log.Printf("protocol error: %s (code: %d)", protoErr.Message, protoErr.Code)

	case errors.As(err, &toolErr):
		log.Printf("tool error: %v", toolErr)

	default:
		log.Printf("unexpected error: %v", err)
	}
}
