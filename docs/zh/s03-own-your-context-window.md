---
title: "s03 · 自己控制上下文窗口（Thread + Event）"
chapter: 3
slug: s03-own-your-context-window
est_read_min: 8
---

# s03 · 自己控制上下文窗口（Thread + Event）

> 教什么：把"输入"从一个字符串升级成事件流。`Thread{Events []Event}` 是整本课程后续所有章节的中央数据结构。

---

## Problem / 问题

s02 让 prompt 显式可见，但 prompt 内嵌的还是一段 raw 用户输入。真实 agent 跑一次会产生 user_input → tool_call → tool_response → tool_call → … 这样一个事件流，每一步都得 LLM 看见。如果 prompt 里只有"最初那段话"，模型既不知道前几步发生了什么，也没法在错误后自我修正。

上游 factor-03 的答案是：**把整个交互历史作为事件序列保存**，序列化后塞进 prompt。这一节我们定下 `Event{Type, Data}` + `Thread{Events []Event}` + `SerializeForLLM()`，后续 9 章在这个结构上加内容（tool 调用、错误事件、人类响应等）。

## Solution / 解决方案

3 个决策：

1. **`Event.Type` 用 const 字符串**而不是 enum int —— JSON 序列化后 LLM 能直接读 `"type":"user_input"`，比看到 `"type":1` 友好得多。
2. **Constructors over literals**：写 `NewUserInputEvent("hi")` 而不是 `Event{Type:"user_input", Data:"hi"}`。constructors 集中在 events.go 里，新加 Event Type 时一眼看全。
3. **`SerializeForLLM()` 默认 JSON indent**：可读性优先。s07 我们会演示 XML 序列化以省 token。

## How It Works / 工作原理

```
   argv ──► NewUserInputEvent("...") ──► Thread{Events:[...]}
                                                  │
                                                  ▼
                                  Thread.SerializeForLLM()  → JSON string
                                                  │
                                                  ▼
                            RenderPrompt(PromptInput{Thread: ...})
                                                  │
                                                  ▼
                            EchoThreadProvider.DetermineNextStep(ctx, rendered)
                                                  │
                                                  ▼
                       NextStep{Intent:"done_for_now",
                                Data:{message:"Thread received with N user_input events."}}
```

核心 30 行（节选自 [`agents/s03-own-your-context-window/thread.go`](https://github.com/Ding-Ye/learn-12-factor-agents/blob/main/agents/s03-own-your-context-window/thread.go)）：

```go
type Event struct {
    Type string      `json:"type"`
    Data interface{} `json:"data"`
}

type Thread struct {
    Events []Event
}

func NewThread(seed ...Event) *Thread {
    t := &Thread{Events: make([]Event, 0, 8)}
    t.Events = append(t.Events, seed...)
    return t
}

func (t *Thread) Append(e Event)            { t.Events = append(t.Events, e) }
func (t *Thread) LastEvent() (Event, bool)  { /* ... */ }
func (t *Thread) SerializeForLLM() string   { /* json.MarshalIndent */ }
```

**3 个非显然之处**：

1. **`make([]Event, 0, 8)` 预分配**：典型 agent 跑 < 8 步，预分配 8 容量避免前几次 append 的 reallocate。微优化但易读，且匹配 12-factor 倡导的"小而专一"。
2. **`Data interface{}` 的 round-trip 问题**：写入时 `Data` 是 `string` / `map[string]any` / 自定义 struct；JSON 解出来都变成 `map[string]any`。这是 Go 的运行时类型擦除，我们 s12 会用 reducer 测试覆盖到。
3. **`SerializeForLLM` 返回 string 不返回 ([]byte, error)**：Provider 接口收 string，保持调用对称。错误兜底成占位 string 让测试 diff 还能看（panicking 会丢测试上下文）。

## What Changed / 与 s02 的变化

```diff
+ events.go      (新建 — Event + 3 constructors)
+ thread.go      (新建 — Thread + Append/LastEvent/SerializeForLLM)
  types.go       (NextStep 不变)
- prompt.go: PromptInput.UserInput
+ prompt.go: PromptInput.Thread
- provider.go: RecordingProvider  
+ provider.go: EchoThreadProvider (数 user_input event 数量)
  main.go        (seed thread → render → provider)
```

语义上的差别：s02 的 provider 看到"一句话 + 模板包装"；s03 看到"JSON 序列化的事件流 + 同一模板包装"。Provider 的接口签名（`string → NextStep`）没变。

## Try It / 动手试一试

```bash
cd agents/s03-own-your-context-window

go test -v ./...

go run . "hello"
# → intent=done_for_now message="Thread received with 1 user_input event."

# 多个 user_input event 不能从 CLI 触发（CLI 只塞一个），但测试覆盖
go test -run TestEchoThreadProvider -v
```

期望输出形态：`intent=done_for_now message="Thread received with 1 user_input event."`

测试覆盖：6 个 PASS（事件追加顺序、序列化稳定性、LastEvent、prompt 渲染、provider 数事件、Event JSON 双向）。

## Upstream Source Reading / 上游源码阅读

```upstream:workshops/2025-07-16/walkthrough/01-agent.py#L1-L26
# Source: workshops/2025-07-16/walkthrough/01-agent.py lines 1-26
# License: Apache 2.0

import json
from typing import Dict, Any, List

AgentResponse = Any

class Event:
    def __init__(self, type: str, data: Any):
        self.type = type
        self.data = data

class Thread:
    def __init__(self, events: List[Dict[str, Any]]):
        self.events = events
    
    def serialize_for_llm(self):
        # can change this to whatever custom serialization you want to do, XML, etc
        return json.dumps(self.events)

def agent_loop(thread: Thread) -> AgentResponse:
    b = get_baml_client()
    next_step = b.DetermineNextStep(thread.serialize_for_llm())
    return next_step
```

**对照阅读要点**：

- **构造器**：Python 用 `Event(type, data)`，我们用 `NewUserInputEvent("...")`。Python 把"任意 type 字符串"留给调用方写，我们用 const 收口。
- **`thread.events` 是 list of dict, not list of Event**：Python 里 Thread 接 dict 列表，事件序列化前是 `Dict[str, Any]`。我们 Go 用 typed Event slice，序列化时不需要先转 dict。
- **`serialize_for_llm()`**：上游也是 JSON 默认（`json.dumps`），注释里提示可换 XML —— 我们 s07 演示 XML。
- **`AgentResponse = Any`**：上游放弃 typing；我们 Go 用 `NextStep` struct 拿回类型安全。这是 Go 相对 Python 的免费红利。
- **未做的部分**：上游 `01-agent.py` 已经有 `agent_loop` —— 但它是 single-shot 的，s05 才引入真 loop。我们 s03 故意只到"一次调用"为止。

**想读更多**：上游 `content/factor-03-own-your-context-window.md` 整篇都值得读，尤其 lines 69-112 讲 XML 序列化的 trade-off。我们 s07 把这条线接上。

---

**下一节预告**：s04 让 Provider 不再总是返回 done_for_now —— 引入 `AddTool` / `SubtractTool` / ... 这一组结构化工具。NextStep 的 Data 开始承载 `{"a":2,"b":3}` 这样的 typed payload，`main.go` 第一次出现 `switch step.Intent` 的 dispatch。
