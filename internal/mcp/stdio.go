package mcp

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"os/exec"
	"sync"
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
		return nil, err

	}

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, err

	}

	if err := cmd.Start(); err != nil {
		return nil, err
	}

	c := &StdioClient{
		cmd:    cmd,
		stdin:  bufio.NewWriter(stdin),
		stdout: bufio.NewScanner(stdout),
		nextID: 1,
	}

	if err := c.initialize(); err != nil {
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
		return err
	}

	if !c.stdout.Scan() {
		return errors.New("no response to initialize")
	}
	log.Printf("received: %s", c.stdout.Text())
	var resp Response

	if err := json.Unmarshal(c.stdout.Bytes(), &resp); err != nil {
		return err
	}

	if resp.Error != nil {
		return fmt.Errorf("initialize error : %s ", resp.Error.Message)
	}

	var initiResult InitializeResult

	if err := json.Unmarshal(resp.Result, &initiResult); err != nil {
		return err
	}

	for name := range initiResult.Capabilities.Tools {

		log.Printf("discovered tool: %s", name)
	}

	initReq := Request{
		JSONRPC: "2.0",
		ID:      c.next(),
		Method:  "initialized",
	}

	if err := c.send(initReq); err != nil {
		return err
	}

	//consumir possivel notificacoes do server
	if c.stdout.Scan() {
		log.Printf("post-init message: %s", c.stdout.Text())

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
		return "", err
	}

	if len(resp.Content) == 0 {
		return "", errors.New("empty file")
	}

	return resp.Content[0].Text, nil
}

func (c *StdioClient) Call(
	ctx context.Context,
	tool string,
	args any,
	result any,
) error {
	req := Request{
		JSONRPC: "2.0",
		ID:      c.next(),
		Method:  "tools/call",
		Params: map[string]any{
			"name":      tool,
			"arguments": args,
		},
	}

	//	var resp struct {
	//		Result InitializeResult `json:"result"`
	//	}

	err := c.send(req)

	if err != nil {
		fmt.Errorf("mcp error: %s", err.Error())
		return err
	}

	if !c.stdout.Scan() {
		return errors.New("No response from MCP Server")
	}

	var rpcResp Response

	if err := json.Unmarshal(c.stdout.Bytes(), &rpcResp); err != nil {
		return err
	}

	if rpcResp.Error != nil {
		return fmt.Errorf("mcp error: %s", rpcResp.Error)
	}

	if result == nil {
		return nil
	}
	return json.Unmarshal(rpcResp.Result, result)

}

func (c *StdioClient) send(req Request) error {

	c.mu.Lock()

	defer c.mu.Unlock()

	b, err := json.Marshal(req)
	log.Printf("sending: %s", string(b))
	if err != nil {
		return err
	}

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
	return c.cmd.Process.Kill()
}

func (c *StdioClient) ListTools(ctx context.Context) ([]Tool, error) {

	req := Request{
		JSONRPC: "2.0",
		ID:      c.next(),
		Method:  "tools/list",
	}

	if err := c.send(req); err != nil {
		return nil, err
	}

	if !c.stdout.Scan() {
		return nil, errors.New("no response from MCP Server")
	}

	var resp struct {
		ID     int `json:"id"`
		Result struct {
			Tools []Tool `json:"tools"`
		} `json:"result"`
	}

	if err := json.Unmarshal(c.stdout.Bytes(), &resp); err != nil {
		return nil, err
	}
	log.Printf("raw mcp response: %s", c.stdout.Text())

	return resp.Result.Tools, nil

}
