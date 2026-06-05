# Aura SDK Demo

This demo demonstrates how external projects can integrate with the Aura SDK.

## Prerequisites

- Go 1.26.1+
- Aura project (for local development)
- LLM provider configured (Ollama, OpenAI, or Anthropic)

## Quick Start

```bash
cd examples/sdk-demo
GOWORK=off go mod tidy
GOWORK=off go run main.go -example basic
```

Note: Since Aura uses Go workspaces, use `GOWORK=off` when building the demo standalone.

## Examples

| Example | Description | Command |
|---------|-------------|---------|
| basic | Minimal SDK integration | `go run main.go -example basic` |
| tool | Custom tool registration | `go run main.go -example tool` |
| confirm | Confirmation handling | `go run main.go -example confirm` |
| stream | Real-time events | `go run main.go -example stream` |
| conversation | Multi-turn chat | `go run main.go -example conversation` |
| timeout | LLM timeout configuration | `go run main.go -example timeout` |
| auto | Auto-approve mode | `go run main.go -example auto` |

## Core Integration Pattern

```go
// 1. Create runtime configuration
cfg := sdk.DefaultRuntimeConfig()

// 2. Create runtime with options
runtime, err := sdk.NewRuntime(cfg,
    sdk.WithMode(sdk.RuntimeModeCLI),
    sdk.WithSessionID("demo-session"),
)

// 3. Initialize
if err := runtime.Initialize(ctx); err != nil { ... }
defer runtime.Shutdown()

// 4. Start and send input via event stream
runtime.Start(ctx)
runtime.SendEvent(ctx, sdk.NewEvent(sdk.EventTypeUserInput, "Hello!", "req-1"))
for ev := range runtime.Events() {
    switch ev.Type() {
    case sdk.EventTypeResponse:
        fmt.Println(ev.Content())
    case sdk.EventTypeDone:
        break
    }
}
runtime.Stop(ctx)
```

## Custom Tools

Implement the Tool interface:

```go
type Tool interface {
    Name() string
    Description() string
    Execute(ctx context.Context, params map[string]any) (*ToolResult, error)
}
```

Register with runtime: `runtime.AddTool(customTool)`

## Configuration

Set LLM provider via environment variables:

```bash
export AURA_LLM_PROVIDER=ollama
export AURA_LLM_BASE_URL=http://localhost:11434
export AURA_LLM_MODEL=qwen3:8b
```

Or programmatically:

```go
cfg.LLM.Provider = "ollama"
cfg.LLM.BaseURL = "http://localhost:11434"
cfg.LLM.Model = "qwen3:8b"
```

### Timeout Configuration

For long-running LLM tasks (complex reasoning, large model inference), configure HTTP timeout:

```go
// Programmatic configuration
cfg.LLM.Timeout = 300 * time.Second  // 5 minutes

// Or via config file (configs/config.yaml):
// llm:
//   timeout: 300s  # 5 minutes
```

**Recommended timeout values:**
- Simple tasks: 120s (default)
- Complex reasoning: 300s-600s (5-10 minutes)
- Large model inference: 600s-1200s (10-20 minutes)

**Note:** LLM timeout is different from context timeout:
- `cfg.LLM.Timeout`: HTTP client timeout for LLM API calls
- `context.WithTimeout`: Overall operation timeout for the entire task