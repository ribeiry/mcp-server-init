# CLAUDE.md - MCP Server Init

## Software Design Document (SDD)

### Visão Geral
Cliente MCP (Model Context Protocol) em Go que se comunica com servidores via stdio usando JSON-RPC 2.0.

### Arquitetura

```
cmd/
  app/
    main.go              → Cliente MCP exemplo (filesystem)
  ollama-server/
    main.go              → Servidor MCP-Ollama (em desenvolvimento)

internal/
  mcp/
    client.go            → Interface Client (contrato)
    protocol.go          → Estruturas JSON-RPC (Request, Response, RPCError)
    stdio.go             → StdioClient (implementação principal)
    errors.go            → Tipos de erro personalizados
    errors_test.go       → Testes de erros (6 testes)
    stdio_test.go        → Testes do cliente (20 testes)
  
  ollama/
    client.go            → Cliente HTTP para API Ollama
```

> **Nota:** Em Go, testes ficam no mesmo pacote que o código testado (sufixo `_test.go`), não em diretório separado.

### Componentes Principais

#### 1. StdioClient
- Gerencia processo filho (servidor MCP)
- Comunicação via stdin/stdout
- Thread-safe com `sync.Mutex` no send

#### 2. Protocolo JSON-RPC 2.0
- Requests: `{"jsonrpc":"2.0","id":N,"method":"...","params":{}}`
- Responses: `{"jsonrpc":"2.0","id":N,"result":{}}` ou `{"error":{"code":N,"message":"..."}}`

#### 3. Handshake MCP
1. `initialize` → negocia protocolo e capacidades
2. `initialized` → confirma conexão estabelecida

#### 4. Métodos Públicos (MCP Cliente)
- `ListTools(ctx)` → lista ferramentas disponíveis
- `ReadFile(ctx, path)` → lê arquivo via ferramenta
- `Call(ctx, tool, args, result)` → chama ferramenta genérica
- `Close()` → graceful shutdown

#### 5. Cliente Ollama (`internal/ollama`)
- `Generate(ctx, model, prompt, system)` → completions
- `Chat(ctx, model, messages)` → conversa com histórico
- `ListModels(ctx)` → lista modelos disponíveis

### Tratamento de Erros

| Tipo | Uso |
|------|-----|
| `ConnectionError` | Pipes, I/O, processo |
| `ProtocolError` | Parse JSON, resposta inválida |
| `ToolError` | Erro retornado pelo servidor |

Pattern de uso:
```go
var connErr *ConnectionError
if errors.As(err, &connErr) { ... }
```

### Requisitos
- Go 1.23.2+
- Node.js com npx (servidores MCP)
- Ollama rodando localmente (para servidor MCP-Ollama)

### Status por Componente

| Componente | Status | Testes |
|------------|--------|--------|
| MCP Cliente (stdio) | ✅ Completo | 26 testes |
| Tratamento de Erros | ✅ Completo | Coberto |
| Ollama Cliente HTTP | ✅ Completo | Pendente |
| MCP Servidor (Ollama) | 🚧 Em desenvolvimento | - |

### Comandos Úteis
```bash
go test ./... -v          # Rodar testes com verbose
go test ./... -race       # Detectar race conditions
go build ./...            # Build completo
go mod tidy               # Limpar dependências
go fmt ./...              # Formatizar código
go vet ./...              # Análise estática
```

---

## O Que NÃO Fazer (Don'ts)

### Código
- ❌ **Não ignore erros de scanner** - sempre verifique `scanner.Err()` após `!Scan()`
- ❌ **Não use `fmt.Errorf` sem wrapping** - use erros tipificados para erros tratáveis
- ❌ **Não mate o processo sem tentar graceful shutdown** - use `os.Interrupt` antes de `Kill()`
- ❌ **Não faça send sem lock** - `stdin` é compartilhado, use `c.mu.Lock()`
- ❌ **Não assuma que `Process` existe** - verifique `c.cmd.Process == nil` no Close
- ❌ **Não esqueça de verificar `ctx.Err()`** - cheque contexto cancelado antes de operações
- ❌ **Não retorne erro genérico** - envolva em `ConnectionError`, `ProtocolError` ou `ToolError`
- ❌ **Não confie em resposta vazia** - valide `len(resp.Result) == 0`

### Testes
- ❌ **Não teste apenas sucesso** - teste contexto cancelado, JSON inválido, erro na resposta
- ❌ **Não use `t.Fatal` sem mensagem** - sempre explique o erro esperado
- ❌ **Não esqueça de verificar tipo de erro** - use `errors.As()` para erros tipificados

### Git/Commit
- ❌ **Não faça commit de .env ou credenciais**
- ❌ **Não esqueça de rodar testes antes de commit** - `go test ./...`
- ❌ **Não faça push sem revisar o diff** - `git diff --stat`

### Estilo Go
- ❌ **Não use `var x type = val`** - use `x := val` quando possível
- ❌ **Não deixe imports unused** - remova imports não utilizados
- ❌ **Não use `panic` em library code** - retorne erro
- ❌ **Não ignore `err` com `_`** - trate ou propague
