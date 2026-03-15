package main

import (
	"context"
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
		log.Fatal(err)
	}

	defer client.Close()

	content, err := client.ReadFile(ctx, "data/test.txt")

	//tools, err := client.ListTools(ctx)

	if err != nil {
		log.Fatal(err)
	}

	log.Print("file content: %s", content)
	//for _, t := range tools {
	//	log.Printf("tool: %s - %s", t.Name, t.Description)
	//}

}
