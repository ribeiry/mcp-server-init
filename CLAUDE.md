# CLAUDE.md - MCP Server Init

## Software Design Document (SDD)

### Visão Geral
Cliente MCP (Model Context Protocol) em Go que se comunica com servidores via stdio usando JSON-RPC 2.0.

### Arquitetura

```
cmd/app/main.go          → Ponto de entrada, handler de erros
internal/mcp/
  client.go              → Interface Client (contrato)
  protocol.go            → Estruturas JSON-RPC (Request, Response, RPCError)
  stdio.go               → StdioClient (implementação principal)
  errors.go              → Tipos de erro personalizados
  errors_test.go         → Testes de erros
  stdio_test.go          → Testes do cliente
```

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

#### 4. Métodos Públicos
- `ListTools(ctx)` → lista ferramentas disponíveis
- `ReadFile(ctx, path)` → lê arquivo via ferramenta
- `Call(ctx, tool, args, result)` → chama ferramenta genérica
- `Close()` → graceful shutdown

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
