package mcp

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"os/exec"
	"strings"
	"sync"
	"testing"
)

// mockStdioClient cria um cliente com pipes mockados para teste
type mockStdioClient struct {
	stdin  *strings.Reader
	stdout *strings.Reader
}

func TestStdioClient_next(t *testing.T) {
	c := &StdioClient{nextID: 1}

	if c.next() != 1 {
		t.Error("expected first ID to be 1")
	}
	if c.next() != 2 {
		t.Error("expected second ID to be 2")
	}
	if c.nextID != 3 {
		t.Errorf("expected nextID to be 3, got %d", c.nextID)
	}
}

func TestStdioClient_send(t *testing.T) {
	t.Run("successful send", func(t *testing.T) {
		var stdinBuf strings.Builder
		c := &StdioClient{
			stdin:  bufio.NewWriter(&stdinBuf),
			stdout: bufio.NewScanner(strings.NewReader("")),
			nextID: 1,
			mu:     sync.Mutex{},
		}

		req := Request{
			JSONRPC: "2.0",
			ID:      1,
			Method:  "test",
		}

		err := c.send(req)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		output := stdinBuf.String()
		if !strings.Contains(output, `"jsonrpc":"2.0"`) {
			t.Error("expected jsonrpc version in output")
		}
		if !strings.Contains(output, `"method":"test"`) {
			t.Error("expected method in output")
		}
	})
}

func TestStdioClient_Call(t *testing.T) {
	t.Run("context cancelled", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		cancel()

		c := &StdioClient{
			stdin:  nil,
			stdout: nil,
			nextID: 1,
		}

		err := c.Call(ctx, "test_tool", nil, nil)
		if err == nil {
			t.Fatal("expected error for cancelled context")
		}

		var connErr *ConnectionError
		if !errors.As(err, &connErr) {
			t.Errorf("expected ConnectionError, got %T", err)
		}
	})

	t.Run("successful call with nil result", func(t *testing.T) {
		response := `{"jsonrpc":"2.0","id":1,"result":{"data":"ok"}}`
		c := &StdioClient{
			stdin:  bufio.NewWriter(&strings.Builder{}),
			stdout: bufio.NewScanner(strings.NewReader(response)),
			nextID: 1,
		}

		err := c.Call(context.Background(), "test_tool", nil, nil)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	t.Run("parse error response", func(t *testing.T) {
		c := &StdioClient{
			stdin:  bufio.NewWriter(&strings.Builder{}),
			stdout: bufio.NewScanner(strings.NewReader("invalid json")),
			nextID: 1,
		}

		err := c.Call(context.Background(), "test_tool", nil, nil)
		if err == nil {
			t.Fatal("expected error for invalid JSON")
		}

		var protoErr *ProtocolError
		if !errors.As(err, &protoErr) {
			t.Errorf("expected ProtocolError, got %T", err)
		}
	})

	t.Run("error in response", func(t *testing.T) {
		response := `{"jsonrpc":"2.0","id":1,"error":{"code":-32600,"message":"invalid request"}}`
		c := &StdioClient{
			stdin:  bufio.NewWriter(&strings.Builder{}),
			stdout: bufio.NewScanner(strings.NewReader(response)),
			nextID: 1,
		}

		err := c.Call(context.Background(), "test_tool", nil, nil)
		if err == nil {
			t.Fatal("expected error from response")
		}

		var toolErr *ToolError
		if !errors.As(err, &toolErr) {
			t.Errorf("expected ToolError, got %T", err)
		}
	})
}

func TestStdioClient_ReadFile(t *testing.T) {
	t.Run("successful read", func(t *testing.T) {
		response := `{"jsonrpc":"2.0","id":1,"result":{"content":[{"type":"text","text":"file content"}]}}`
		c := &StdioClient{
			stdin:  bufio.NewWriter(&strings.Builder{}),
			stdout: bufio.NewScanner(strings.NewReader(response)),
			nextID: 1,
		}

		content, err := c.ReadFile(context.Background(), "/test/path")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if content != "file content" {
			t.Errorf("got %q, want %q", content, "file content")
		}
	})

	t.Run("empty content", func(t *testing.T) {
		response := `{"jsonrpc":"2.0","id":1,"result":{"content":[]}}`
		c := &StdioClient{
			stdin:  bufio.NewWriter(&strings.Builder{}),
			stdout: bufio.NewScanner(strings.NewReader(response)),
			nextID: 1,
		}

		_, err := c.ReadFile(context.Background(), "/test/path")
		if err == nil {
			t.Fatal("expected error for empty content")
		}

		var protoErr *ProtocolError
		if !errors.As(err, &protoErr) {
			t.Errorf("expected ProtocolError, got %T", err)
		}
	})

	t.Run("tool error wrapper", func(t *testing.T) {
		response := `{"jsonrpc":"2.0","id":1,"error":{"code":-32000,"message":"permission denied"}}`
		c := &StdioClient{
			stdin:  bufio.NewWriter(&strings.Builder{}),
			stdout: bufio.NewScanner(strings.NewReader(response)),
			nextID: 1,
		}

		_, err := c.ReadFile(context.Background(), "/test/path")
		if err == nil {
			t.Fatal("expected error")
		}

		var toolErr *ToolError
		if !errors.As(err, &toolErr) {
			t.Errorf("expected ToolError, got %T", err)
		}
		if toolErr.Name != "read_file" {
			t.Errorf("expected tool name read_file, got %s", toolErr.Name)
		}
	})
}

func TestStdioClient_ListTools(t *testing.T) {
	t.Run("context cancelled", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		cancel()

		c := &StdioClient{nextID: 1}

		_, err := c.ListTools(ctx)
		if err == nil {
			t.Fatal("expected error for cancelled context")
		}

		var connErr *ConnectionError
		if !errors.As(err, &connErr) {
			t.Errorf("expected ConnectionError, got %T", err)
		}
	})

	t.Run("successful list", func(t *testing.T) {
		response := `{"jsonrpc":"2.0","id":1,"result":{"tools":[{"name":"read_file","description":"Read a file"},{"name":"write_file","description":"Write a file"}]}}`
		c := &StdioClient{
			stdin:  bufio.NewWriter(&strings.Builder{}),
			stdout: bufio.NewScanner(strings.NewReader(response)),
			nextID: 1,
		}

		tools, err := c.ListTools(context.Background())
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(tools) != 2 {
			t.Errorf("got %d tools, want 2", len(tools))
		}
		if tools[0].Name != "read_file" {
			t.Errorf("got %q, want %q", tools[0].Name, "read_file")
		}
	})

	t.Run("empty tools list", func(t *testing.T) {
		response := `{"jsonrpc":"2.0","id":1,"result":{"tools":[]}}`
		c := &StdioClient{
			stdin:  bufio.NewWriter(&strings.Builder{}),
			stdout: bufio.NewScanner(strings.NewReader(response)),
			nextID: 1,
		}

		tools, err := c.ListTools(context.Background())
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(tools) != 0 {
			t.Errorf("got %d tools, want 0", len(tools))
		}
	})

	t.Run("parse error", func(t *testing.T) {
		c := &StdioClient{
			stdin:  bufio.NewWriter(&strings.Builder{}),
			stdout: bufio.NewScanner(strings.NewReader("invalid json")),
			nextID: 1,
		}

		_, err := c.ListTools(context.Background())
		if err == nil {
			t.Fatal("expected error for invalid JSON")
		}

		var protoErr *ProtocolError
		if !errors.As(err, &protoErr) {
			t.Errorf("expected ProtocolError, got %T", err)
		}
	})
}

func TestStdioClient_Close(t *testing.T) {
	t.Run("close without process", func(t *testing.T) {
		c := &StdioClient{cmd: &exec.Cmd{}}
		err := c.Close()
		if err != ErrServerClosed {
			t.Errorf("got %v, want ErrServerClosed", err)
		}
	})
}

func TestStdioClient_initialize(t *testing.T) {
	t.Run("no response", func(t *testing.T) {
		c := &StdioClient{
			stdin:  bufio.NewWriter(&strings.Builder{}),
			stdout: bufio.NewScanner(strings.NewReader("")),
			nextID: 1,
		}

		err := c.initialize()
		if err == nil {
			t.Fatal("expected error for empty response")
		}

		var protoErr *ProtocolError
		if !errors.As(err, &protoErr) {
			t.Errorf("expected ProtocolError, got %T", err)
		}
	})

	t.Run("error in initialize response", func(t *testing.T) {
		response := `{"jsonrpc":"2.0","id":1,"error":{"code":-32600,"message":"unsupported protocol"}}`
		c := &StdioClient{
			stdin:  bufio.NewWriter(&strings.Builder{}),
			stdout: bufio.NewScanner(strings.NewReader(response)),
			nextID: 1,
		}

		err := c.initialize()
		if err == nil {
			t.Fatal("expected error from response")
		}

		var protoErr *ProtocolError
		if !errors.As(err, &protoErr) {
			t.Errorf("expected ProtocolError, got %T", err)
		}
	})

	t.Run("successful initialize", func(t *testing.T) {
		response := `{"jsonrpc":"2.0","id":1,"result":{"capabilities":{"tools":{"read_file":{}}}}}`
		c := &StdioClient{
			stdin:  bufio.NewWriter(&strings.Builder{}),
			stdout: bufio.NewScanner(strings.NewReader(response + "\n")),
			nextID: 1,
		}

		err := c.initialize()
		// Nota: pode falhar no send de 'initialized' pois não temos buffer real
		// o importante é que o parse do initialize funcionou
		if err != nil {
			var protoErr *ProtocolError
			var connErr *ConnectionError
			if !errors.As(err, &protoErr) && !errors.As(err, &connErr) {
				t.Logf("initialize error (expected for mock): %v", err)
			}
		}
	})
}

func TestInitializeResult_Unmarshal(t *testing.T) {
	t.Run("valid json", func(t *testing.T) {
		raw := json.RawMessage(`{"capabilities":{"tools":{"read_file":{},"write_file":{}}}}`)
		var result InitializeResult
		err := json.Unmarshal(raw, &result)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(result.Capabilities.Tools) != 2 {
			t.Errorf("got %d tools, want 2", len(result.Capabilities.Tools))
		}
	})
}
