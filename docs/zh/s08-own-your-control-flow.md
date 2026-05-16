---
title: "s08 · 自己控制流程 (loop / break / branch)"
chapter: 8
slug: s08-own-your-control-flow
est_read_min: 8
---

# s08 · 自己控制流程 (loop / break / branch)

> 教什么：把 s07 隐含的 "if intent ... else if ..." 链显式化成 `Action` enum + `ControlFlow` 函数。加新 intent 没分支就跑 `ActionEscalate`，靠 `KnownIntents` + 测试守护"加 intent 必须改 ControlFlow"。

---

## Problem / 问题

s07 的 `RunAgent` 里有这样的隐式分支：

- tool.Execute 返回 ErrHumanContact → 早退（不算错）
- intent == done_for_now → 早退
- 其它 → loop

读代码时这些分支散在 loop 里、和 error handling 混在一起。新加一个 intent（比如 s09 的 `error` 自我修复）很容易漏判某条路径。

上游 factor-08 的口号是 **own your control flow**：分支应当**显式 + 数据驱动 + 可枚举**。

## Solution / 解决方案

3 个决策：

1. **`Action` 是 enum**：`ActionLoop` / `ActionBreak` / `ActionFinish` / `ActionEscalate`。加新分支 = 新加常量 + ControlFlow 加 case。
2. **`ControlFlow(thread, next, registry) Action` 是纯函数**：无 I/O 无副作用。loop 里 dispatch 一次拿 Action，再 switch on Action 做事。
3. **`KnownIntents()` + exhaustiveness 测试**：列出所有已知 intent，测试遍历它们断言 ControlFlow 不返回 `ActionInvalid`。新加 intent 但忘了改 KnownIntents → 测试 still pass；但新加 intent 在 `ControlFlow` switch 无 case → ControlFlow 走 default → `ActionEscalate`，运行时立刻被发现。

## How It Works / 工作原理

```
   RunAgent loop:
       ┌────────────────────────────────┐
       │ provider.DetermineNextStep     │
       │   ── append tool_call to thread │
       │   ── action = ControlFlow(...) │
       │                                 │
       │ switch action {                │
       │   case ActionFinish: return     │
       │   case ActionBreak:  return     │
       │   case ActionLoop:              │
       │     ── tool.Execute             │
       │     ── append tool_response     │
       │   case ActionEscalate:          │
       │     ── append error event       │
       │     ── return with error        │
       │ }                               │
       └────────────────────────────────┘
```

核心 30 行（节选自 [`controlflow.go`](https://github.com/Ding-Ye/learn-12-factor-agents/blob/main/agents/s08-own-your-control-flow/controlflow.go)）：

```go
type Action int
const (
    ActionInvalid Action = iota
    ActionLoop
    ActionBreak
    ActionFinish
    ActionEscalate
)

func ControlFlow(_ *Thread, next NextStep, registry Registry) Action {
    switch next.Intent {
    case IntentDoneForNow:
        return ActionFinish
    case IntentRequestApproval, IntentRequestMoreInformation:
        return ActionBreak
    default:
        if _, ok := registry[next.Intent]; ok {
            return ActionLoop
        }
        return ActionEscalate
    }
}
```

**3 个非显然之处**：

1. **`ControlFlow` 不接 ctx，不返回 error**：纯函数。这让测试可以一行写完每个分支（`got := ControlFlow(nil, NextStep{Intent: x}, r)`）。所有 side effect 在 RunAgent 的 switch on Action 里。
2. **`ActionEscalate` 不 panic**：upstream 也可能这样设计（unknown intent → panic / fast-fail）。我们选 escalate 是因为生产场景里 LLM emit 错 intent 是普通情况（hallucination），不该崩溃。
3. **`KnownIntents()` 不是必需的，但很有用**：手写 list 是 "documentation as test" —— 看一眼就知道 ControlFlow 支持哪些 intent。Go 没有 sealed enum，这个手段是最接近的等价物。

## What Changed / 与 s07 的变化

```diff
+ controlflow.go (新建 — Action enum + ControlFlow + KnownIntents)
- loop.go: if/else 早退分支
+ loop.go: switch on ControlFlow(...) 返回值
+ events.go: + EventTypeError + NewErrorEvent (s09 会大用)
- 5 tests
+ 7 tests (含 exhaustiveness 测试)
```

语义上的差别：s07 是隐式分支；s08 把分支固定到 `Action` enum，扩展性收紧（加 intent 必须显式标注哪个 Action）。

## Try It / 动手试一试

```bash
cd agents/s08-own-your-control-flow

go test -v ./...

go run .
# Final thread has 6 events:
#   [0] user_input
#   [1] tool_call (add)
#   [2] tool_response
#   [3] tool_call (multiply)
#   [4] tool_response
#   [5] tool_call (request_approval, breaks loop)
```

## Upstream Source Reading / 上游源码阅读

```upstream:workshops/2025-07-16/walkthrough/07-agent.py#L38-L80
# Source: workshops/2025-07-16/walkthrough/07-agent.py lines 38-80
# License: Apache 2.0

def agent_loop(thread: Thread) -> AgentResponse:
    b = get_baml_client()
    while True:
        next_step = b.DetermineNextStep(thread.serialize_for_llm())
        thread.events.append({"type": next_step.intent, "data": next_step})
        
        if next_step.intent == "done_for_now":
            return next_step
        elif next_step.intent == "request_more_information":
            clarification = clarification_handler(next_step.message)
            thread.events.append({"type": "clarification_response", "data": clarification})
        elif next_step.intent == "fetch_issues":
            issues = await linear_client.issues()
            thread.events.append({"type": "fetch_issues_result", "data": issues})
        elif next_step.intent == "add":
            result = next_step.a + next_step.b
            thread.events.append({"type": "tool_response", "data": result})
        # ... 更多 elif
```

**对照阅读要点**：

- **`if/elif` 链 vs `Action` enum**：上游每加一种 intent 就加一个 elif；我们把"判断"提到 `ControlFlow` 一个函数里，"执行"留在 loop。一种关注点分离。
- **上游"sync tool" 和 "async tool" 混在 if/elif**：upstream `fetch_issues` 是 `await`，`add` 是 sync 计算，都进同一个 elif；我们把"是否要 Execute"交给 `Action`（`ActionLoop` 一定要 Execute），更整洁。
- **上游没 `ActionEscalate`**：默认行为是 `else` 分支不存在 → 落空 → loop 下一轮直接重发 prompt 给 LLM。我们 Go 端 escalate + append error event，让"unknown intent"变成一次明确的失败而不是无尽 retry。
- **测试 exhaustiveness**：上游没法做（Python 没枚举）；Go 可以靠 `KnownIntents` 自检。
- **上游使用 `await`**：upstream Python 跑在 asyncio；我们 Go 用 goroutine + 同步代码。

**想读更多**：上游 `content/factor-08-own-your-control-flow.md:27-68` 把"为什么 control flow 不能交给 framework"讲得很清楚。

---

**下一节预告**：s09 让 tool error 不再终止 agent —— 改成 append `error` event 到 thread，让 LLM 看到错误信息并自我修复。连续 N 个 error → escalate。
