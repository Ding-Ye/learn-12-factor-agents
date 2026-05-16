---
title: "s04 · 工具即结构化输出"
chapter: 4
slug: s04-tools-are-structured-outputs
est_read_min: 8
---

# s04 · 工具即结构化输出

> 教什么：LLM 输出从"自由文本"升级到"typed tool struct"。NextStep.Data 第一次承载 typed payload，main.go 第一次出现 `switch step.Intent` dispatch。

---

## Problem / 问题

s03 把上下文做对了，但 LLM 还是只能返回 done_for_now。真实 agent 要让模型说出"add 2 + 3"或"call this API"这样的**typed action**，并且我们的代码能立刻 dispatch 到正确的执行函数。如果让 LLM 返回 free text，每次都要写 parser、处理"3 加 2"和"two plus three"——本质上是把每个 prompt 都退化成 NLP 任务。

上游 factor-04 的回答：**每个工具是一个 typed struct**。LLM 看到 schema、emit 符合 schema 的 JSON、我们 type-switch 就执行。这一节我们定下 `Tool` 接口 + 5 个 concrete tool（add/subtract/multiply/divide/done_for_now）+ `Registry` 集中查表。

## Solution / 解决方案

3 个决策：

1. **`Tool` 接口只有两个方法**：`Intent() string` 返回 discriminator，`Execute(ctx, json.RawMessage) (any, error)` 执行。极简表面，让"加新工具"成本最低（写个 struct，实现 2 个方法）。
2. **`MathPayload` 共享给 4 个 math 工具**：upstream BAML 写了 4 个独立 class（AddTool/SubtractTool/MultiplyTool/DivideTool），字段都一样。我们 Go 端共享 `MathPayload{A,B}`，discriminator 留在 NextStep.Intent 上。
3. **`Registry` 是 `map[string]Tool` 而不是 `[]Tool`**：dispatch 是 O(1)；新加 tool 只改 `DefaultRegistry()` 一处。

## How It Works / 工作原理

```
   argv ──► ScriptedProvider.DetermineNextStep ──► NextStep{Intent:"add", Data:{a,b}}
                                                              │
                                                              ▼
                                          Registry.Lookup(step.Intent) ──► AddTool{}
                                                              │
                                                              ▼
                                          AddTool.Execute(ctx, step.Data) ──► 5.0
                                                              │
                                                              ▼
                                                    print result
```

核心 30 行（节选自 [`agents/s04-tools-are-structured-outputs/tools.go`](https://github.com/Ding-Ye/learn-12-factor-agents/blob/main/agents/s04-tools-are-structured-outputs/tools.go)）：

```go
type Tool interface {
    Intent() string
    Execute(ctx context.Context, payload json.RawMessage) (any, error)
}

type AddTool struct{}

func (AddTool) Intent() string { return IntentAdd }
func (AddTool) Execute(_ context.Context, payload json.RawMessage) (any, error) {
    var p MathPayload
    if err := json.Unmarshal(payload, &p); err != nil {
        return nil, fmt.Errorf("decode add payload: %w", err)
    }
    return p.A + p.B, nil
}

type Registry map[string]Tool

func DefaultRegistry() Registry { /* ... */ }
func (r Registry) Lookup(intent string) (Tool, error) { /* ... */ }
```

**3 个非显然之处**：

1. **`Tool.Execute` 不接 Thread**：tools 是纯函数 `(input → output)`。Thread 累积逻辑是 loop 的事（s05 引入）。这个分层让 tool 可以单测，也让 s10 的 sub-agent 可以共享 tool 池。
2. **`done_for_now` 不实现 Tool 接口**：它是"loop 退出信号"而不是动作。main.go 在 dispatch 前 special-case 处理。s05 我们会用 `ControlFlow` 把这种 special case 显式化。
3. **DivideTool 的错误信息逐字匹配上游**："Error: Division by zero" 是 upstream `05-agent.py:51-52` 原话。s09 引入错误进 thread 时，这个字符串会成为 LLM 学到的"我犯了什么错"的依据。

## What Changed / 与 s03 的变化

```diff
+ types.go: MathPayload, intent constants
+ tools.go (新建 — Tool interface + 4 math tools + Registry)
- provider.go: EchoThreadProvider
+ provider.go: ScriptedProvider (根据 input 关键词 emit 不同 intent)
- main.go: 单一渲染
+ main.go: dispatch（Registry.Lookup + Tool.Execute）
- 6 tests
+ 7 tests
```

语义上的差别：s03 的 provider 永远 done_for_now；s04 的 provider 第一次基于 input emit 非 done 的 NextStep。dispatch 还是单步 —— 没有 loop。

## Try It / 动手试一试

```bash
cd agents/s04-tools-are-structured-outputs

go test -v ./...

go run . "add 2 and 3"
# → intent=add payload={"a":2,"b":3} result=5

go run . "multiply 4 and 6"
# → intent=multiply payload={"a":4,"b":6} result=24

go run . "say hi"
# → intent=done_for_now message="Nothing to do."
```

## Upstream Source Reading / 上游源码阅读

```upstream:workshops/2025-07-16/walkthrough/05-agent.baml#L1-L37
// Source: workshops/2025-07-16/walkthrough/05-agent.baml lines 1-37
// License: Apache 2.0

class AddTool {
  intent "add"
  a int | float
  b int | float
}

class SubtractTool {
  intent "subtract"
  a int | float
  b int | float
}

class MultiplyTool {
  intent "multiply"
  a int | float
  b int | float
}

class DivideTool {
  intent "divide"
  a int | float
  b int | float
}

class DoneForNow {
  intent "done_for_now"
  message string
}

function DetermineNextStep(thread: string)
    -> DoneForNow | AddTool | SubtractTool | MultiplyTool | DivideTool {
    client Qwen3
    prompt #" ... "#
}
```

**对照阅读要点**：

- **BAML 的 union return** vs **我们的 NextStep**：BAML 把"返回 5 种 tool 之一"写在函数签名里（`-> A | B | C | D | E`），LLM 端有 schema 强制；我们 Go 端用 `NextStep{Intent, Data}` + Registry 在运行时校验。功能等价，但 BAML 的编译期保证更强。
- **`a int | float` vs `A float64`**：BAML 区分 int 和 float（union 类型）；Go 我们统一 float64 简化 —— 上游 Python 也通过 `MathPayload` 一种类型解决。
- **每个 class 都重复 `a` `b` 字段**：BAML 的 class 不能直接继承字段；我们 Go 把 4 个 math tools 共用 `MathPayload`。
- **divide-by-zero 的处理**：上游 `05-agent.py:51-52` 在执行端 catch 后返回字符串 "Error: Division by zero"；我们 Go 返回 `error`，main.go 决定怎么呈现。s09 才会把这种错误进 thread。
- **`client Qwen3`**：BAML 配 Baseten 上的 Qwen3 32B。我们 Phase G 会演示 swap 到 OpenAI/Anthropic。

**想读更多**：上游 `content/factor-04-tools-are-structured-outputs.md:11-50` 讲为什么"tool 就是返回 typed struct"是关键 abstraction —— 推荐看完。

---

**下一节预告**：s05 把 main.go 的 single-step dispatch 升级成 agent loop —— `RunAgent(thread, provider, tools)` 反复调用 provider → execute → append，直到 done_for_now 才退出。Thread 第一次承载多个 tool_call / tool_response 事件。
