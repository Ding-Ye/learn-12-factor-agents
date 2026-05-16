---
title: "Multi-model support · Anthropic / OpenAI / Baseten in practice"
slug: multi-model
est_read_min: 9
---

# Multi-model support

> Every `Provider` in the 12 chapters is a stub. This addendum shows how to swap stubs for real LLMs: full Anthropic implementation, OpenAI mapping notes, and an integration test pattern.

---

## Why "one-line swap" works

All twelve chapters interact with the LLM through one interface:

```go
type Provider interface {
    DetermineNextStep(ctx context.Context, serialized string) (NextStep, error)
}
```

`ScriptedSequenceProvider` satisfies it, `EchoProvider` satisfies it, and future `AnthropicProvider` / `OpenAIProvider` / `BasetenProvider` will too. This is factor-01 in code: **the LLM is a deterministic input/output function with the same interface as the rest of your code**.

## Anthropic provider — full implementation

Production-ready minimal version (drop into any chapter's `provider.go`):

```go
type AnthropicProvider struct {
    APIKey string
    Model  string  // e.g. "claude-3-5-sonnet-20241022"
    Tools  []ToolSchema
    client *http.Client
}

type anthropicRequest struct {
    Model     string         `json:"model"`
    MaxTokens int            `json:"max_tokens"`
    System    string         `json:"system,omitempty"`
    Messages  []anthropicMsg `json:"messages"`
    Tools     []ToolSchema   `json:"tools,omitempty"`
}

type anthropicMsg struct {
    Role    string                  `json:"role"`
    Content []anthropicContentBlock `json:"content"`
}

type anthropicContentBlock struct {
    Type  string          `json:"type"`            // "text" | "tool_use" | "tool_result"
    Text  string          `json:"text,omitempty"`
    ID    string          `json:"id,omitempty"`
    Name  string          `json:"name,omitempty"`
    Input json.RawMessage `json:"input,omitempty"`
}

type anthropicResponse struct {
    Content    []anthropicContentBlock `json:"content"`
    StopReason string                  `json:"stop_reason"`
}

type ToolSchema struct {
    Name        string                 `json:"name"`
    Description string                 `json:"description"`
    InputSchema map[string]interface{} `json:"input_schema"`
}

func NewAnthropicProvider(apiKey, model string, tools []ToolSchema) *AnthropicProvider {
    return &AnthropicProvider{
        APIKey: apiKey,
        Model:  model,
        Tools:  tools,
        client: &http.Client{Timeout: 120 * time.Second},
    }
}

func (a *AnthropicProvider) DetermineNextStep(ctx context.Context, serialized string) (NextStep, error) {
    req := anthropicRequest{
        Model:     a.Model,
        MaxTokens: 1024,
        System:    "You are an agent. Pick the next tool to call. " +
                   "Return tool_use blocks for actions, or use 'done_for_now' tool to finish.",
        Messages: []anthropicMsg{{
            Role: "user",
            Content: []anthropicContentBlock{{
                Type: "text",
                Text: "Thread so far:\n" + serialized,
            }},
        }},
        Tools: a.Tools,
    }
    body, _ := json.Marshal(req)

    httpReq, err := http.NewRequestWithContext(ctx, "POST",
        "https://api.anthropic.com/v1/messages", bytes.NewReader(body))
    if err != nil {
        return NextStep{}, err
    }
    httpReq.Header.Set("Content-Type", "application/json")
    httpReq.Header.Set("x-api-key", a.APIKey)
    httpReq.Header.Set("anthropic-version", "2023-06-01")

    resp, err := a.client.Do(httpReq)
    if err != nil {
        return NextStep{}, err
    }
    defer resp.Body.Close()
    raw, _ := io.ReadAll(resp.Body)
    if resp.StatusCode/100 != 2 {
        return NextStep{}, fmt.Errorf("anthropic API %d: %s", resp.StatusCode, string(raw))
    }

    var out anthropicResponse
    if err := json.Unmarshal(raw, &out); err != nil {
        return NextStep{}, fmt.Errorf("decode response: %w (body=%s)", err, raw)
    }

    // Find first tool_use block.
    for _, blk := range out.Content {
        if blk.Type == "tool_use" {
            return NextStep{Intent: blk.Name, Data: blk.Input}, nil
        }
    }
    // No tool_use → done_for_now with concatenated text.
    var text string
    for _, blk := range out.Content {
        if blk.Type == "text" {
            text += blk.Text
        }
    }
    data, _ := json.Marshal(DoneForNowPayload{Message: text})
    return NextStep{Intent: "done_for_now", Data: data}, nil
}
```

## OpenAI version (Chat Completions API)

OpenAI's function-calling API differs slightly, but the mapping to `NextStep` is the same:

```go
type OpenAIProvider struct {
    APIKey string
    Model  string  // "gpt-4o" / "gpt-4o-mini"
    Tools  []OpenAIToolSchema
    client *http.Client
}

type OpenAIToolSchema struct {
    Type     string                 `json:"type"` // "function"
    Function map[string]interface{} `json:"function"`
}

// Request:
// POST https://api.openai.com/v1/chat/completions
// {
//   "model": "...",
//   "messages": [...],
//   "tools": [...],
//   "tool_choice": "auto"
// }
//
// Response: choices[0].message.tool_calls[0] is the next tool;
// without tool_calls, treat message.content as done_for_now message.
```

Full implementation is left as [an extension exercise](https://github.com/humanlayer/12-factor-agents/discussions).

## Baseten / Qwen3 (upstream default)

Upstream's BAML targets Baseten-hosted Qwen3 32B. Baseten's API is OpenAI-compatible — point `OpenAIProvider` at `https://bridge.baseten.co/<deployment_id>/sync/v1/chat/completions`.

## Swap steps (using s05 as the example)

1. `cd agents/s05-unify-execution-state`
2. Paste the `AnthropicProvider` code into a new file `provider_anthropic.go` (same `package main`)
3. Hand-write `ToolSchema` entries alongside `tools.go`:

```go
var Schemas = []ToolSchema{
    {Name: "add", Description: "add two numbers",
     InputSchema: map[string]interface{}{
         "type": "object",
         "properties": map[string]any{
             "a": map[string]any{"type":"number"},
             "b": map[string]any{"type":"number"},
         },
         "required": []string{"a","b"},
     }},
    {Name: "multiply", Description: "multiply two numbers", InputSchema: ...},
    {Name: "done_for_now", Description: "finish", InputSchema: ...},
}
```

4. Edit `main.go`:

```diff
- provider := &ScriptedSequenceProvider{Steps: []NextStep{...}}
+ provider := NewAnthropicProvider(
+     os.Getenv("ANTHROPIC_API_KEY"),
+     "claude-3-5-sonnet-20241022",
+     Schemas,
+ )
```

5. Run:

```bash
export ANTHROPIC_API_KEY=sk-ant-...
go run . "add 5 and 3, then multiply by 2"
```

Expected: a real LLM returns `NextStep{intent:"add",...}` and the loop terminates in 3 turns.

## Integration test (`//go:build integration`)

Keep real-provider tests out of the unit-test pool:

```go
//go:build integration

package main

import (
    "context"
    "os"
    "testing"
)

func TestRunAgent_RealAnthropic(t *testing.T) {
    key := os.Getenv("ANTHROPIC_API_KEY")
    if key == "" {
        t.Skip("ANTHROPIC_API_KEY not set")
    }
    provider := NewAnthropicProvider(key, "claude-3-5-haiku-20241022", Schemas)
    thread := NewThread(NewUserInputEvent("add 5 and 3"))
    final, err := RunAgent(context.Background(), thread, provider, DefaultRegistry())
    if err != nil {
        t.Fatalf("RunAgent: %v", err)
    }
    // Assert final is done_for_now and mentions 8 in the message.
}

// Run with:
//   go test -tags=integration ./...
```

CI skips the integration tag by default. To schedule it: inject the key via GitHub secret + cron workflow.

## Provider comparison

| Dimension | Anthropic | OpenAI | Baseten/Qwen3 |
|---|---|---|---|
| Endpoint | `api.anthropic.com/v1/messages` | `api.openai.com/v1/chat/completions` | `bridge.baseten.co/.../v1/chat/completions` (OpenAI-compat) |
| Tool schema | top-level `tools` field | `tools[].function` | same as OpenAI |
| Tool response | `content[].tool_use{name, input}` | `choices[0].message.tool_calls[]` | same as OpenAI |
| Streaming | `?stream=true` via SSE | `?stream=true` via SSE | same as OpenAI |
| Multi-tool in one response | yes | yes | yes |
| Cost (Sonnet vs 4o-mini vs Qwen3) | $$$$ | $$ | $ |
| Latency (median) | ~2-4s | ~1-2s | varies |

## Common pitfalls

1. **Loose JSON schemas**: Anthropic's `input_schema` and OpenAI's `parameters` need OpenAPI compatibility. Missing `required` → the model may skip fields.
2. **`max_tokens` too small**: default 1024 isn't enough for long tool chains. 4096+ for real workloads.
3. **Rate limits**: pair with s09's self-heal — rate-limit errors enter the thread, and the LLM (often) responds with "let me try again" naturally.
4. **API version drift**: Anthropic uses `anthropic-version: 2023-06-01` header; OpenAI versions are implicit in model names. Pin versions.

---

With this page you can swap stubs for real LLMs and run end-to-end. Combined with the integration tests, that's the starting point for a real deployment.
