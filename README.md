# MCP Server Init - Go

Cliente MCP (Model Context Protocol) em Go para comunicação com servidores via stdio.

## Propósito

Este projeto é um **lab de estudos** para aprender e experimentar a implementação do protocolo MCP em Go. O objetivo é entender como:

- Implementar comunicação JSON-RPC 2.0 via stdio
- Conectar-se a servidores MCP (como `@modelcontextprotocol/server-filesystem`)
- Gerenciar handshake (`initialize`/`initialized`)
- Chamar ferramentas remotas (`tools/call`, `tools/list`, `read_file`)
- Tratar erros de forma estruturada

## Requisitos

- **Go** 1.23.2 ou superior
- **Node.js** com `npx` (para executar servidores MCP)

## Estrutura do Projeto

```
.
├── cmd/
│   └── app/
│       └── main.go           # Ponto de entrada
├── internal/
│   └── mcp/
│       ├── client.go         # Interface Client
│       ├── errors.go         # Tipos de erro personalizados
│       ├── protocol.go       # Estruturas JSON-RPC
│       ├── stdio.go          # Implementação StdioClient
│       ├── errors_test.go    # Testes de erros
│       └── stdio_test.go     # Testes do cliente
├── go.mod
└── README.md
```

## Tipos de Erro

O projeto define erros tipificados para melhor tratamento:

| Erro | Descrição |
|------|-----------|
| `ConnectionError` | Falhas de conexão (pipes, stdin/stdout) |
| `ProtocolError` | Erros de protocolo JSON-RPC |
| `ToolError` | Erros em ferramentas específicas |
| `ErrServerClosed` | Servidor já foi fechado |
| `ErrEmptyResponse` | Resposta vazia do servidor |
| `ErrInvalidResponse` | Resposta inválida |

## Uso

```go
ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
defer cancel()

client, err := mcp.NewFileSystemClient(ctx)
if err != nil {
    log.Fatal(err)
}
defer client.Close()

// Listar ferramentas
tools, err := client.ListTools(ctx)

// Ler arquivo
content, err := client.ReadFile(ctx, "/path/to/file")
```

## Testes

```bash
go test ./internal/mcp/... -v
```

## Aprendizados

- Implementação de cliente JSON-RPC 2.0
- Comunicação via stdio com processos filhos
- Graceful shutdown de processos
- Pattern de erros wrapping com `errors.As` e `errors.Is`
- Testes unitários com mocks de I/O

## Recursos

- [Model Context Protocol Documentation](https://modelcontextprotocol.io/)
- [JSON-RPC 2.0 Specification](https://www.jsonrpc.org/specification)
