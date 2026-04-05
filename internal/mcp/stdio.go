package mcp

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"os/exec"
	"sync"
	"time"
)

type StdioClient struct {
	cmd    *exec.Cmd
	stdin  *bufio.Writer
	stdout *bufio.Scanner
	mu     sync.Mutex
	nextID int
}

type InitializeResult struct {
	Capabilities struct {
		Tools map[string]any `json:"tools"`
	} `json:"capabilities"`
}

func NewFileSystemClient(ctx context.Context) (*StdioClient, error) {
	cmd := exec.CommandContext(
		ctx,
		"npx",
		"@modelcontextprotocol/server-filesystem",
		".",
	)

	stdin, err := cmd.StdinPipe()
	if err != nil {
		return nil, &ConnectionError{Op: "get stdin pipe", Err: err}
	}

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, &ConnectionError{Op: "get stdout pipe", Err: err}
	}

	if err := cmd.Start(); err != nil {
		return nil, &ConnectionError{Op: "start command", Err: err}
	}

	c := &StdioClient{
		cmd:    cmd,
		stdin:  bufio.NewWriter(stdin),
		stdout: bufio.NewScanner(stdout),
		nextID: 1,
	}

	if err := c.initialize(); err != nil {
		c.forceKill()
		return nil, err
	}

	return c, nil
}

func (c *StdioClient) initialize() error {
	req := Request{
		JSONRPC: "2.0",
		ID:      c.next(),
		Method:  "initialize",
		Params: map[string]any{
			"protocolVersion": "2024-11-05",
			"capabilities":    map[string]any{},
			"clientInfo": map[string]string{
				"name":    "mcp-server-init",
				"version": "0.1.0",
			},
		},
	}

	if err := c.send(req); err != nil {
		return &ConnectionError{Op: "send initialize", Err: err}
	}

	if !c.stdout.Scan() {
		if err := c.stdout.Err(); err != nil {
			return &ConnectionError{Op: "read initialize response", Err: err}
		}
		return &ProtocolError{Op: "initialize", Message: "no response from server"}
	}

	raw := c.stdout.Bytes()
	log.Printf("received: %s", raw)

	var resp Response
	if err := json.Unmarshal(raw, &resp); err != nil {
		return &ProtocolError{Op: "parse initialize", Message: err.Error()}
	}

	if resp.Error != nil {
		return &ProtocolError{Op: "initialize", Message: resp.Error.Message, Code: resp.Error.Code}
	}

	if len(resp.Result) == 0 {
		return &ProtocolError{Op: "initialize", Message: "empty result"}
	}

	var initResult InitializeResult
	if err := json.Unmarshal(resp.Result, &initResult); err != nil {
		return &ProtocolError{Op: "parse initialize result", Message: err.Error()}
	}

	for name := range initResult.Capabilities.Tools {
		log.Printf("discovered tool: %s", name)
	}

	initReq := Request{
		JSONRPC: "2.0",
		ID:      c.next(),
		Method:  "initialized",
	}

	if err := c.send(initReq); err != nil {
		return &ConnectionError{Op: "send initialized", Err: err}
	}

	// consumir possiveis notificacoes do server
	if c.stdout.Scan() {
		log.Printf("post-init message: %s", c.stdout.Text())
	} else if err := c.stdout.Err(); err != nil {
		log.Printf("warning: error reading post-init notification: %v", err)
	}

	return nil
}

func (c *StdioClient) ReadFile(ctx context.Context, path string) (string, error) {
	var resp struct {
		Content []struct {
			Type string `json:"type"`
			Text string `json:"text"`
		} `json:"content"`
	}

	err := c.Call(ctx, "read_file", map[string]any{
		"path": path,
	}, &resp)
	if err != nil {
		return "", &ToolError{Name: "read_file", Err: err}
	}

	if len(resp.Content) == 0 {
		return "", &ProtocolError{Op: "read_file", Message: "empty content"}
	}

	return resp.Content[0].Text, nil
}

func (c *StdioClient) Call(
	ctx context.Context,
	tool string,
	args any,
	result any,
) error {
	if ctx.Err() != nil {
		return &ConnectionError{Op: "call " + tool, Err: ctx.Err()}
	}

	req := Request{
		JSONRPC: "2.0",
		ID:      c.next(),
		Method:  "tools/call",
		Params: map[string]any{
			"name":      tool,
			"arguments": args,
		},
	}

	if err := c.send(req); err != nil {
		return &ConnectionError{Op: "send " + tool, Err: err}
	}

	if !c.stdout.Scan() {
		if err := c.stdout.Err(); err != nil {
			return &ConnectionError{Op: "read " + tool + " response", Err: err}
		}
		return &ProtocolError{Op: "call " + tool, Message: "no response from server"}
	}

	var rpcResp Response
	if err := json.Unmarshal(c.stdout.Bytes(), &rpcResp); err != nil {
		return &ProtocolError{Op: "parse " + tool + " response", Message: err.Error()}
	}

	if rpcResp.Error != nil {
		return &ToolError{Name: tool, Err: &ProtocolError{Op: "execute", Message: rpcResp.Error.Message, Code: rpcResp.Error.Code}}
	}

	if result == nil {
		return nil
	}

	if err := json.Unmarshal(rpcResp.Result, result); err != nil {
		return &ProtocolError{Op: "parse " + tool + " result", Message: err.Error()}
	}

	return nil
}

func (c *StdioClient) send(req Request) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	b, err := json.Marshal(req)
	if err != nil {
		return fmt.Errorf("marshal request: %w", err)
	}

	log.Printf("sending: %s", b)

	if _, err := c.stdin.Write(append(b, '\n')); err != nil {
		return err
	}

	return c.stdin.Flush()
}

func (c *StdioClient) next() int {
	id := c.nextID
	c.nextID++
	return id
}

func (c *StdioClient) Close() error {
	return c.terminate()
}

func (c *StdioClient) terminate() error {
	if c.cmd.Process == nil {
		return ErrServerClosed
	}

	// tenta graceful shutdown primeiro
	err := c.cmd.Process.Signal(os.Interrupt)
	if err != nil {
		return c.forceKill()
	}

	// aguarda processo terminar
	done := make(chan error, 1)
	go func() {
		done <- c.cmd.Wait()
	}()

	select {
	case <-time.After(5 * time.Second):
		return c.forceKill()
	case err := <-done:
		return err
	}
}

func (c *StdioClient) forceKill() error {
	if c.cmd.Process == nil {
		return ErrServerClosed
	}
	return c.cmd.Process.Kill()
}

func (c *StdioClient) ListTools(ctx context.Context) ([]Tool, error) {
	if ctx.Err() != nil {
		return nil, &ConnectionError{Op: "list tools", Err: ctx.Err()}
	}

	req := Request{
		JSONRPC: "2.0",
		ID:      c.next(),
		Method:  "tools/list",
	}

	if err := c.send(req); err != nil {
		return nil, &ConnectionError{Op: "send tools/list", Err: err}
	}

	if !c.stdout.Scan() {
		if err := c.stdout.Err(); err != nil {
			return nil, &ConnectionError{Op: "read tools/list response", Err: err}
		}
		return nil, &ProtocolError{Op: "tools/list", Message: "no response from server"}
	}

	var resp struct {
		ID     int `json:"id"`
		Result struct {
			Tools []Tool `json:"tools"`
		} `json:"result"`
	}

	if err := json.Unmarshal(c.stdout.Bytes(), &resp); err != nil {
		return nil, &ProtocolError{Op: "parse tools/list", Message: err.Error()}
	}

	log.Printf("raw mcp response: %s", c.stdout.Text())

	return resp.Result.Tools, nil
}
