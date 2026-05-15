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

// 4. Process input - returns event channel
events, err := runtime.Process(ctx, "Hello!")
for ev := range events {
    switch ev.Type() {
    case sdk.EventTypeResponse:
        fmt.Println(ev.Content())
    case sdk.EventTypeDone:
        break
    }
}
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