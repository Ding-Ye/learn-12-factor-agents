---
title: "s09 · 错误进上下文，模型自愈"
chapter: 9
slug: s09-compact-errors
est_read_min: 8
---

# s09 · 错误进上下文，模型自愈

> 教什么：tool 报错不再让 agent 终止。错误进 thread 变成 `error` event，LLM 下一轮看到后自我修复。连续 ≥3 次错误才 escalate。

---

## Problem / 问题

s08 的 RunAgent 见到 tool error 就退出并把 error 抛出。但真实场景里：

- 模型 emit `divide(10, 0)` → tool 报 "Division by zero" → 如果直接退出，agent 还没尝试别的
- 模型可能在下一轮看到错误后改成 `divide(10, 2)` 自我修复

上游 factor-09 的论点：**错误是数据**。把错误塞进 thread 作为 event，让 LLM 看；如果模型连续 N 次错（=陷入"打死还要踩"），再 escalate。

## Solution / 解决方案

3 个决策：

1. **`NewErrorEvent` 是一种新 event type**：放在 tool_call 后面，无 tool_response。LLM 下次见到这对 (tool_call, error)，prompt 里就有自我修复的根据。
2. **`ConsecutiveErrors(thread)` 数尾部 error**：从最后向前数 (tool_call, error) 对；第一个非 error event 截断。靠 thread 推导，无 side state。
3. **`SafeExecute` 用 `recover()` 兜底 panic**：buggy tool panic 不应该崩 agent。panic → 转 error → 同样的 self-heal 流程。

## How It Works / 工作原理

```
   RunAgent step N:
       provider → next (e.g., divide(10, 0))
       append tool_call(next)
       tool.Execute(divide, {a:10,b:0}) → error "Division by zero"
       ┌──────────────────────────────────────────┐
       │ append NewErrorEvent(err.Error())        │
       │ if ConsecutiveErrors(thread) >= 3 → escalate │
       │ else: continue                           │
       └──────────────────────────────────────────┘
       
   step N+1:
       provider sees thread including the error event
       returns NextStep{intent: "divide", a: 10, b: 2}  ← LLM self-corrected
       tool.Execute → 5  (success!)
       append tool_response(5)
```

核心 30 行（节选自 [`agents/s09-compact-errors/thread.go`](https://github.com/Ding-Ye/learn-12-factor-agents/blob/main/agents/s09-compact-errors/thread.go) + `loop.go`）：

```go
func ConsecutiveErrors(t *Thread) int {
    count := 0
    for i := len(t.Events) - 1; i >= 0; i-- {
        switch t.Events[i].Type {
        case EventTypeError:
            count++
        case EventTypeToolCall:
            continue   // 配对的 tool_call 不打断 streak
        default:
            return count   // 非 error 非 tool_call event 切断 streak
        }
    }
    return count
}

// inside RunAgent:
result, err := SafeExecute(ctx, tool, next.Data)
if err != nil {
    thread.Append(NewErrorEvent(err.Error()))
    if ConsecutiveErrors(thread) >= MaxConsecutiveErrors {
        return thread, fmt.Errorf("escalated after %d consecutive errors", MaxConsecutiveErrors)
    }
    continue
}
thread.Append(NewToolResponseEvent(result))
```

**3 个非显然之处**：

1. **`ConsecutiveErrors` 从尾向前**：只看"最近的 streak"。如果中间有过一次成功（tool_response），streak 重置。
2. **`SafeExecute` 用闭包 + named return**：`defer func() { ... }()` 里改 `err` 变量必须用 named return。`(result any, err error)` 是 Go 里 panic recovery 的惯用法。
3. **escalate 是 explicit error return**：不是 panic、不是 silent break。HTTP handler 看见 error 可以决定是 500 还是 502 + 留 thread 给人工看。

## What Changed / 与 s08 的变化

```diff
+ types.go: + MaxConsecutiveErrors = 3 + IntentDivide
+ thread.go: + ConsecutiveErrors helper
+ tools.go: + DivideTool, SafeExecute (panic recover)
- loop.go: tool error 立即 return
+ loop.go: append error event, check counter, continue or escalate
- 7 tests
+ 6 tests (含 panic 测试 + escalation 测试)
```

语义上的差别：s08 的失败=终止；s09 的失败=机会。LLM 第一次有了"看到错误然后改"的能力。

## Try It / 动手试一试

```bash
cd agents/s09-compact-errors

go test -v ./...

go run .
# Final thread has 6 events:
#   [0] user_input: divide 10 by 0, then 10 by 2
#   [1] tool_call: intent=divide
#   [2] error: Error: Division by zero
#   [3] tool_call: intent=divide
#   [4] tool_response: 5
#   [5] tool_call: intent=done_for_now
# Consecutive errors at end: 0
```

第一次 divide(10, 0) 报错，进 error event；第二次 divide(10, 2)=5；最后 done。整段 trace 里有错有恢复有结束。

## Upstream Source Reading / 上游源码阅读

```upstream:workshops/2025-07-16/walkthrough/03-agent.py#L21-L35
# Source: workshops/2025-07-16/walkthrough/03-agent.py lines 21-35
# License: Apache 2.0

# Note: this is a slightly different (Pythonic) shape than our Go port.
# Upstream's compact-error pattern is documented but not always present
# in every walkthrough file; the canonical reference is the markdown.

try:
    result = await handle_next_step(thread, next_step)
    consecutive_errors = 0
except Exception as e:
    consecutive_errors += 1
    if consecutive_errors < 3:
        thread.events.append({"type": "error", "data": format_error(e)})
    else:
        break  # escalate
```

**对照阅读要点**：

- **`try/except` vs `SafeExecute + recover`**：上游 Python `except Exception` 捕获 panic / 异常都走同一路径；Go 把 panic 和 error 分开，`SafeExecute` 把 panic 转 error 统一处理。
- **`consecutive_errors` 是局部变量** vs **`ConsecutiveErrors(thread)` 派生函数**：上游存在 loop 内的整数变量；我们靠 thread 推导。后者好处：HTTP resume 后（thread 从 store 取出来）counter 自动正确，不需要 serialize 出来。
- **`format_error(e)` vs `err.Error()`**：上游单独函数清理 traceback；Go 的 err.Error() 默认就是简短消息，更适合给 LLM 看。
- **escalate 上游 `break` 退出 loop 但不抛 error** vs **我们返回 error**：trade-off 取决于谁来发现 "agent 卡住了"。上游靠 loop 后逻辑检查；我们靠 caller 检查 return error。
- **缺的部分**：上游 factor-09 markdown 里建议 "compact" error 而不是塞完整 traceback —— 因为 LLM 上下文有限。我们 Go 端用 `err.Error()` 已经是 compact 的，但生产场景可以再做摘要（去掉文件路径 / 调用栈）。

**想读更多**：`content/factor-09-compact-errors.md:10-59` 通篇都讲"为什么不是 fast-fail"——值得一读。

---

**下一节预告**：s10 引入 sub-agent 编排：一个 `Orchestrator` 把任务拆给 `CalcAgent`（数学）和 `SummaryAgent`（自然语言总结）。每个 sub-agent 有自己独立的 thread —— 上下文小、可独立测、可并行。
