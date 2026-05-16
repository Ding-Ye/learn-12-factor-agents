---
title: "s02 · 自己控制 prompt"
chapter: 2
slug: s02-own-your-prompts
est_read_min: 7
---

# s02 · 自己控制 prompt

> 教什么：让 prompt 不再藏在框架的暗处。每一个 token 都来自你写的模板；测试可以断言"模板真的跑过了"。

---

## Problem / 问题

s01 把 wire format 定下了（`NextStep` + `Provider`），但 prompt 还是个谜——EchoProvider 无视输入返回固定值。在真实代码里，模型表现的可预测性几乎全部取决于 prompt：错一个 token，输出就漂；改一个角色标记，模型就乱。

上游 factor-02 的论点是：**prompt 是代码**。不该让 langchain / crewai 之类的框架替你拼 system message，因为出问题时你既看不见拼出来的最终 prompt，也调不动其中任何一段。s02 把这个原则落到 Go：用 `text/template` 显式渲染，并通过 `prompt_hash` 让测试断言渲染确实跑过。

## Solution / 解决方案

3 个决策：

1. **`promptTemplate` 是 Go 字符串常量**，结构对齐上游 BAML 的 system / user 双段式。读这一行代码就知道发往 LLM 的全部内容。
2. **`RenderPrompt(PromptInput) (string, error)`** 是唯一渲染入口。`PromptInput` 是 struct（不是 map），新字段缺失会变 compile error。
3. **`RecordingProvider`** 替代 s01 的 EchoProvider。它把收到的 prompt 存到 `LastSeen`，并在返回的 `done_for_now` 消息里嵌入 `prompt_hash=...`。测试可以拿这个 hash 对比期望值——template 改了 hash 就变。

## How It Works / 工作原理

```
   argv ─► RenderPrompt(PromptInput) ─► rendered string (SYSTEM:/USER:)
                                                   │
                                                   ▼
                          RecordingProvider.DetermineNextStep(ctx, rendered)
                                          │
                                          ├─ stores rendered to .LastSeen
                                          ▼
                          NextStep{Intent:"done_for_now",
                                   Data:{message:"Acknowledged. prompt_hash=<sha8>"}}
                                          │
                                          ▼
                                  renderNextStep ─► stdout
```

核心 30 行（节选自 [`agents/s02-own-your-prompts/prompt.go`](https://github.com/Ding-Ye/learn-12-factor-agents/blob/main/agents/s02-own-your-prompts/prompt.go)）：

```go
const promptTemplate = `SYSTEM:
You are a helpful assistant that responds to the user's message.

USER:
You are given the following thread of events:
{{ .UserInput }}

What should the next step be?
`

type PromptInput struct {
    UserInput string
}

func RenderPrompt(in PromptInput) (string, error) {
    t, err := template.New("agent-prompt").Parse(promptTemplate)
    if err != nil { return "", fmt.Errorf("parse template: %w", err) }
    var buf bytes.Buffer
    if err := t.Execute(&buf, in); err != nil {
        return "", fmt.Errorf("execute template: %w", err)
    }
    return buf.String(), nil
}
```

**3 个非显然之处**：

1. **`PromptInput` 是 struct 不是 map** —— `map[string]any` 会让模板里写错字段也不报错（只是渲染成空）。struct 让 `{{ .UserInpit }}` 这种 typo 在 `Execute` 时显式失败。
2. **`PromptHash` 只取前 8 字节** —— sha256 全量 32 字节对人眼太长。前 8 字节 = 16 hex 字符，足够区分但 readable。这个折衷的 trade-off 是冲突概率（~10^-19，对教学场景可忽略）。
3. **`RecordingProvider.LastSeen` 是 exported field** —— 故意的。测试需要 inspect 渲染后的完整 prompt（hash 不匹配时定位差异）。生产代码会把它改成私有 + getter，但这里 readability > encapsulation。

## What Changed / 与 s01 的变化

```diff
+ prompt.go               (新建 — RenderPrompt + PromptHash)
  types.go                (NextStep + DoneForNowPayload 不变)
- provider.go: EchoProvider  
+ provider.go: RecordingProvider (.LastSeen + 嵌 hash)
  main.go                 (多一步 RenderPrompt before provider 调用)
- 3 tests
+ 6 tests
```

语义上的差别：s01 的 provider 是"无视输入"的 stub；s02 的 provider 是"必须看到完整 rendered prompt"的 stub。后续每一章的 provider 都会接受这个约定。

## Try It / 动手试一试

```bash
cd agents/s02-own-your-prompts

go test -v ./...

go run . "add 5 and 3"
# → intent=done_for_now message="Acknowledged. prompt_hash=<16-hex-chars>"

go run . "different input"
# hash 不同 — 证明 RenderPrompt 真的渲染了不同内容
```

期望输出形态：

```
intent=done_for_now message="Acknowledged. prompt_hash=03b6f1c2afe0945b"
```

不同 input 渲染出来的 prompt 不同，hash 就不同。改 `promptTemplate` 重跑测试也会 fail —— `TestPromptHash_Stable` 锁住"同一 input 出同一 hash"，但 `TestRenderPrompt_HasRoleMarkers` 检查的是结构。

## Upstream Source Reading / 上游源码阅读

上游把 prompt 写在 BAML 文件里：

```upstream:workshops/2025-07-16/walkthrough/01-agent.baml#L11-L27
// Source: workshops/2025-07-16/walkthrough/01-agent.baml lines 11-27
function DetermineNextStep(
    thread: string
) -> DoneForNow {
    client Qwen3
    prompt #"
        {{ _.role("system") }}
        You are a helpful assistant that responds to the user's message.

        {{ _.role("user") }}
        You are given the following thread of events:
        {{ thread }}

        What should the next step be?
    "#
}
```

**对照阅读要点**：

- **BAML `{{ _.role("system") }}` vs Go `SYSTEM:`**：BAML 这个标记会被编译器翻译成 Anthropic API 期望的 `messages[].role` 字段。我们 Go 端没接真 API 之前，用文本 `SYSTEM:` / `USER:` 就够。Phase G 接 Anthropic / OpenAI provider 时，这两个标记会被解析成真正的 messages 数组。
- **BAML 的 `{{ thread }}` vs Go 的 `{{ .UserInput }}`**：上游已经引入了 Thread 概念（thread 是 string，由 `serialize_for_llm` 返回）；我们 s02 还没引入 Thread，所以暂时用 UserInput。s03 会切换成 `{{ .Thread }}`。
- **`prompt #"..."# `**：BAML 的多行字符串语法，相当于 Go 的 `` ` `` raw string。
- **`client Qwen3`**：BAML 把 LLM 客户端配置放在 prompt 旁边。我们把这件事推到 Provider 实现（s02 是 RecordingProvider，Phase G 才换成真客户端）。
- **缺的部分**：上游 BAML 还能用 `{{ ctx.output_format }}` 自动注入 typed-output 的 JSON Schema —— 我们 s04 才会手写这个 schema，目前不需要。

**想读更多**：上游 `content/factor-02-own-your-prompts.md:14-91` 有完整的"为什么 prompt 要自己写"论述。值得读完，是整个 12-factor 论调的精华段落之一。

---

**下一节预告**：s03 把"输入"从一个字符串升级成 `Thread{Events []Event}`。Provider 收到的不再是 `UserInput`，而是 `json.Marshal(thread.Events)` 的结果。这是 12-factor 里"自己拥有 context window"那条原则的落地。
