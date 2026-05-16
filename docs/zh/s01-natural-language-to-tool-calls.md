---
title: "s01 · 最小 agent 原语：自然语言到工具调用"
chapter: 1
slug: s01-natural-language-to-tool-calls
est_read_min: 8
---

# s01 · 最小 agent 原语：自然语言到工具调用

> 教什么：把"LLM 怎么把自然语言变成代码能 dispatch 的东西"这件事讲清楚 —— 不是讲怎么 prompt，而是讲 **它返回什么格式，我们怎么消费**。一切后续章节都建立在这一节定下的 `NextStep` 类型上。

---

## Problem / 问题

自然语言到代码之间，缺少一个**双方都同意的中间层**。直接让 LLM 返回字符串"add 2 and 3"，意味着我们得写正则、写解析器、还得处理"3 加 2"和"two plus three"这些花活——本质上是把每个 prompt 都退化成 NLP 任务。

上游 12-factor 的答案是：让 LLM 返回**结构化的 typed object**（BAML 的 class、Python 的 dataclass、Go 的 struct），让 deterministic 代码用 type switch 决定下一步。这一节我们不立刻接上真 LLM —— 先把"返回的结构长什么样"定好，并写一个 stub provider 跑通整个链路。这是后续 11 章的地基。

## Solution / 解决方案

3 个关键决策点：

1. **Provider 是一个接口**，而不是一个 Anthropic / OpenAI SDK 调用。整本课程的所有章节，从 stub 到真 LLM，都只看到 `Provider.DetermineNextStep(ctx, serialized) (NextStep, error)` 这一个签名。
2. **NextStep 是带 discriminator 的 tagged union**：`Intent string` 是 tag，`Data json.RawMessage` 是 payload。Intent 决定要 unmarshal 成哪种 payload struct。这是 BAML `class XTool { intent "x"; ... }` 模式的 Go 翻译。
3. **EchoProvider 是 deterministic stub**：它无视输入永远返回同一个 `done_for_now`。这让本章和后续每章的测试都不依赖网络、不依赖 API key、可重放。

## How It Works / 工作原理

```
   argv ─► main ─► EchoProvider.DetermineNextStep(ctx, input)
                                        │
                                        ▼
                            NextStep{Intent, Data}
                                        │
                                        ▼
                  renderNextStep(step)  ──── switch step.Intent
                                              ├── "done_for_now" → 解 DoneForNowPayload
                                              └── default        → 打印 raw JSON
                                        │
                                        ▼
                                     stdout
```

核心 30 行（节选自 [`agents/s01-natural-language-to-tool-calls/provider.go`](https://github.com/Ding-Ye/learn-12-factor-agents/blob/main/agents/s01-natural-language-to-tool-calls/provider.go)）：

```go
type Provider interface {
    DetermineNextStep(ctx context.Context, serialized string) (NextStep, error)
}

type EchoProvider struct{}

const DefaultMessage = "Hello! How can I assist you today?"

func (EchoProvider) DetermineNextStep(_ context.Context, _ string) (NextStep, error) {
    data, err := json.Marshal(DoneForNowPayload{Message: DefaultMessage})
    if err != nil {
        return NextStep{}, err
    }
    return NextStep{
        Intent: "done_for_now",
        Data:   data,
    }, nil
}
```

**3 个非显然之处**：

1. **`Data` 是 `json.RawMessage` 而不是 `interface{}`** —— RawMessage 保留了原始字节，意味着 marshal/unmarshal 两次的结果完全一样。如果用 `interface{}`，第二次 marshal 时 `map[string]interface{}` 的 key 顺序就不稳定了。s06 把 NextStep 通过 HTTP 传出去时，这个稳定性变得很重要。
2. **`DetermineNextStep` 收 `serialized string`** —— 不是 `Thread`，不是 `[]Message`。s01 这个参数被忽略；但提前定下 string，意味着 s03 引入 Thread 后只要改实现不需要改签名。Provider 永远只看到"已经序列化好的一段文本"。
3. **error 返回值看似多余** —— EchoProvider 永不出错。但接口签名要为未来真 provider（s02+，最终 Phase G 的 Anthropic/OpenAI）留位置；超时、HTTP 错误、JSON 解析错都得有路出来。

## What Changed / 与 上一节 的变化

s01 是 bootstrap 章节，没有上一节可比。建立的是 baseline：

- `types.go` —— `NextStep` + `DoneForNowPayload`
- `provider.go` —— `Provider` interface + `EchoProvider`
- `main.go` —— CLI 入口、`renderNextStep` 分发
- `provider_test.go` —— 5 个测试覆盖 wire format

后面 11 章都在这 4 个文件的形态上演化。

## Try It / 动手试一试

```bash
cd agents/s01-natural-language-to-tool-calls

# 跑测试（无网络依赖）
go test -v ./...

# 默认问候
go run . "hello"

# 任何输入都返回同一个 done_for_now
go run . "add 5 and 3, then multiply by 2"
```

期望输出形态：

```
intent=done_for_now message="Hello! How can I assist you today?"
```

测试输出：5 个 PASS，覆盖 (1) intent 永远是 done_for_now、(2) message 是 canonical 字符串、(3) NextStep JSON 双向稳定、(4) renderNextStep 标准路径、(5) renderNextStep fallback 路径。

## Upstream Source Reading / 上游源码阅读

上游的对应代码在 **BAML 端**（`workshops/2025-07-16/walkthrough/01-agent.baml`）和 **Python 端**（`workshops/2025-07-16/walkthrough/01-agent.py`）。BAML 是上游用来生成 typed LLM output 的 DSL —— 它本质上做的事情和我们 Go 的 `NextStep{Intent, Data}` 一样，只是用 codegen 而不是手写。

```upstream:workshops/2025-07-16/walkthrough/01-agent.baml#L1-L27
// Source: workshops/2025-07-16/walkthrough/01-agent.baml lines 1-27
// 这是 BAML 的语法：声明一个返回 typed union 的函数

class DoneForNow {
  intent "done_for_now"      // ← 字面量类型，相当于 Go 里的 `Intent string`
  message string              // ← 就是我们的 DoneForNowPayload.Message
}

// 这个 BAML 函数声明 = 我们的 Provider.DetermineNextStep 接口签名
function DetermineNextStep(
    thread: string             // ← 注意：上游 thread 也是 string,
) -> DoneForNow {              //    印证我们 Provider 收 string 的选择
    client Qwen3               // ← BAML 端配置 LLM 客户端
    prompt #"
        {{ _.role("system") }}
        You are a helpful assistant that responds to the user's message.
        {{ _.role("user") }}
        You are given the following thread of events:
        {{ thread }}
    "#
}
```

```upstream:workshops/2025-07-16/walkthrough/01-agent.py#L23-L26
# Source: workshops/2025-07-16/walkthrough/01-agent.py lines 23-26
# Python 端的调用点 —— 等价于我们 main.go 里的 EchoProvider.DetermineNextStep 调用
def agent_loop(thread: Thread) -> AgentResponse:
    b = get_baml_client()
    next_step = b.DetermineNextStep(thread.serialize_for_llm())
    return next_step
```

**对照阅读要点**：

- **BAML 的 codegen vs 我们手写的 struct**：上游用 BAML 编译出 typed Python class；我们手写 `DoneForNowPayload` + `json.RawMessage`。功能等价，但 BAML 多一个 build step。我们的选择让 Go 学习者少装一个工具链。
- **`prompt` 块**：上游把 prompt 直接写进 BAML 文件；我们 s01 还没引入 prompt 模板，s02 才会把它移到 Go `text/template`。
- **`client Qwen3`**：上游把 LLM 客户端配置写在 BAML 里；我们 s01 用 stub，Phase G 才会引入真客户端。
- **`thread: string` 参数**：上游和我们一样用 string —— 印证"Provider 永远看到 serialized 文本"这个决定是上游的真实做法，不是为了简化 Go 而做的妥协。
- **未做的部分**：上游 `01-agent.py` 里还有 `Event` / `Thread` / `serialize_for_llm`——这些归 s03 引入。s01 保持最小：只要 NextStep。

**想读更多**：从 `workshops/2025-07-16/walkthrough/01-agent.baml` 入手，跟着 `DetermineNextStep` 进 `01-agent.py`，再看 `01-main.py` 怎么调用。这条线在我们的课程里对应 s01 → s03 → s04 的演化。

---

**下一节预告**：s02 把"prompt 是 framework 黑盒"这件事撕开 —— 我们用 Go `text/template` 显式渲染 prompt，让 provider 收到的不再是裸输入而是渲染好的文本。EchoProvider 升级为"返回 prompt hash"以便测试断言渲染确实跑过。
