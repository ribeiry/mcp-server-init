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
│   ├── app/
│   │   └── main.go              # Cliente MCP exemplo
│   └── ollama-server/
│       └── main.go              # Servidor MCP-Ollama (em desenvolvimento)
├── internal/
│   ├── mcp/
│   │   ├── client.go            # Interface Client
│   │   ├── errors.go            # Tipos de erro personalizados
│   │   ├── errors_test.go       # Testes de erros (6 testes)
│   │   ├── protocol.go          # Estruturas JSON-RPC
│   │   ├── stdio.go             # Implementação StdioClient
│   │   └── stdio_test.go        # Testes do cliente (20 testes)
│   └── ollama/
│       └── client.go            # Cliente HTTP API Ollama
├── go.mod
├── README.md
└── CLAUDE.md
```

> **Nota:** Em Go, os testes (`*_test.go`) ficam no mesmo diretório que o código testado, não em pasta separada. Isso permite testar funções privadas e facilita navegação.

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
# Rodar todos os testes
go test ./... -v

# Rodar testes do pacote MCP
go test ./internal/mcp/... -v

# Rodar com detecção de race conditions
go test ./... -race
```

### Cobertura de Testes

| Pacote | Arquivos de Teste | Testes |
|--------|-------------------|--------|
| `internal/mcp` | `errors_test.go`, `stdio_test.go` | 26 testes |
| `internal/ollama` | (em desenvolvimento) | - |

## Aprendizados

- Implementação de cliente JSON-RPC 2.0
- Comunicação via stdio com processos filhos
- Graceful shutdown de processos
- Pattern de erros wrapping com `errors.As` e `errors.Is`
- Testes unitários com mocks de I/O
- Cliente HTTP para API Ollama (Generate, Chat, ListModels)

## Recursos

- [Model Context Protocol Documentation](https://modelcontextprotocol.io/)
- [JSON-RPC 2.0 Specification](https://www.jsonrpc.org/specification)
