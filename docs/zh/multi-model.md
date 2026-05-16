---
title: "多模型支持 · Anthropic / OpenAI / Baseten 实战"
slug: multi-model
est_read_min: 9
---

# 多模型支持

> 12 章读完后，所有 `Provider` 实现都是 stub。本文教你怎么把 stub 换成真 LLM：Anthropic、OpenAI、Baseten 三种 provider 的实现 + 1 个接入示例 + 一个集成测试。

---

## 为什么"swap 一行就行"

整本课程 12 章都通过同一个 interface 与 LLM 交互：

```go
type Provider interface {
    DetermineNextStep(ctx context.Context, serialized string) (NextStep, error)
}
```

`ScriptedSequenceProvider` 满足它，`EchoProvider` 满足它，未来的 `AnthropicProvider` / `OpenAIProvider` / `BasetenProvider` 也要满足它。这是 12-factor 的 factor-01 思想：**LLM 是一个 deterministic input/output 函数，与你的业务代码用同样的接口对接**。

## Anthropic provider 完整实现

下面是 production-ready 的最小版（适合放进 s05 / s06 任何一章的 `provider.go`）：

```go
// AnthropicProvider calls https://api.anthropic.com/v1/messages.
// Add tools[] to the request and parse tool_use content blocks into NextStep.
type AnthropicProvider struct {
    APIKey string
    Model  string  // e.g. "claude-3-5-sonnet-20241022"
    Tools  []ToolSchema // see "schema" section below
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
        return NextStep{}, fmt.Errorf("decode anthropic response: %w (body=%s)", err, raw)
    }

    // Anthropic returns either text or tool_use blocks. Find first tool_use.
    for _, blk := range out.Content {
        if blk.Type == "tool_use" {
            return NextStep{
                Intent: blk.Name,
                Data:   blk.Input,
            }, nil
        }
    }
    // No tool_use → treat as done_for_now with concatenated text.
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

## OpenAI 版本（Chat Completions API）

OpenAI 的 function-calling API 略不同，但映射到我们 NextStep 的逻辑一致：

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

// 请求：
// POST https://api.openai.com/v1/chat/completions
// {
//   "model": "...",
//   "messages": [...],
//   "tools": [...],
//   "tool_choice": "auto"
// }
//
// 返回的 choices[0].message.tool_calls[0] 就是下一步 tool；
// 没有 tool_calls 时拿 content 作为 done_for_now message。
```

完整代码做成 [issue/discussion 形式留给 extension exercise](https://github.com/humanlayer/12-factor-agents/discussions)。

## Baseten / Qwen3（上游 default）

上游 BAML 默认用 Baseten 跑 Qwen3 32B。Baseten 的 API 兼容 OpenAI 的 chat completions，把 `OpenAIProvider` 的 endpoint 换成 `https://bridge.baseten.co/<deployment_id>/sync/v1/chat/completions` 即可。

## 把 stub 换成真 provider 的步骤（以 s05 为例）

1. `cd agents/s05-unify-execution-state`
2. 把上面的 `AnthropicProvider` 完整代码贴到一个新文件 `provider_anthropic.go`（注意 package main 同名）
3. 在 `tools.go` 旁加一个 `ToolSchema` 列表（手写每个 tool 的 JSON schema）：

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

4. 修改 `main.go`：

```diff
- provider := &ScriptedSequenceProvider{Steps: []NextStep{...}}
+ provider := NewAnthropicProvider(
+     os.Getenv("ANTHROPIC_API_KEY"),
+     "claude-3-5-sonnet-20241022",
+     Schemas,
+ )
```

5. 跑：

```bash
export ANTHROPIC_API_KEY=sk-ant-...
go run . "add 5 and 3, then multiply by 2"
```

期望：真 LLM 返回 NextStep{intent:"add",...}，loop 跑 3 步终止。

## 集成测试（`//go:build integration`）

把真 provider 测试和 unit test 分开：

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
    // 检查最后是不是 done_for_now，message 提到 8
    if !IsDone(final.LastEvent()) { /* ... */ }
}

// 跑：
//   go test -tags=integration ./...
```

CI 默认不跑 integration tag —— `.github/workflows/go.yml` 的 `go test ./...` 不带 tag。如果想 schedule 跑，用 GitHub secret 注入 API key + cron workflow。

## 三个 provider 的对比

| 维度 | Anthropic | OpenAI | Baseten/Qwen3 |
|---|---|---|---|
| Endpoint | `api.anthropic.com/v1/messages` | `api.openai.com/v1/chat/completions` | `bridge.baseten.co/.../v1/chat/completions` (OpenAI 兼容) |
| Tool schema | top-level `tools` 字段 | `tools[].function` | 同 OpenAI |
| Tool response 形式 | `content[].tool_use{name, input}` | `choices[0].message.tool_calls[]` | 同 OpenAI |
| 流式 | `?stream=true` 用 SSE | `?stream=true` 用 SSE | 同 OpenAI |
| Multi-tool in single response | 支持 | 支持 | 支持 |
| Cost (Sonnet vs 4o-mini vs Qwen3) | $$$$ | $$ | $ |
| Latency (median) | ~2-4s | ~1-2s | varies |

## 常见坑

1. **JSON schema 不严格**：Anthropic 的 `input_schema` 要 OpenAPI compatible；OpenAI 的 `parameters` 同。漏 `required` 数组 → 模型可能不传字段。
2. **`max_tokens` 不够**：默认 1024 对长 tool chain 不够。real 调用建议 4096+。
3. **rate limit**：用 s09 的 self-heal 流程 —— rate limit 错误进 thread，retry 时 LLM 自然 sleep（往往会自己说"let me try again"）。
4. **API version drift**：Anthropic 用 `anthropic-version: 2023-06-01` header；OpenAI 通过 model name 隐含版本。Pin version。

---

完成此页后，你已经能把 stub 换成真 LLM，跑端到端。配合 Phase G 的 integration tests，这就是 production deployment 的起点。
