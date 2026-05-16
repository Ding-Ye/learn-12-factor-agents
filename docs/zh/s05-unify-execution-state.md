---
title: "s05 · 统一执行状态与业务状态"
chapter: 5
slug: s05-unify-execution-state
est_read_min: 9
---

# s05 · 统一执行状态与业务状态

> 教什么：所有"执行中信息"都在 Thread.Events 里。"agent 跑到第几步了 / 错了几次 / 在等谁"这些问题都靠查 thread 回答 —— 不需要额外的 state 表。

---

## Problem / 问题

s04 完成了"单步 dispatch"，但真实 agent 要"反复 dispatch 直到完成"。常见反模式是引入"执行状态"对象（current_step、is_waiting、error_count），和"业务状态"（user_input、tool_results）分开存。两份状态很容易不同步：业务状态记的是 step 3 完成，执行状态记的是 step 2。

上游 factor-05 的答案：**只保留一份状态——Thread.Events**。query 这个 list 就能算出所有派生信息。我们 s05 把这个原则落到 Go：`RunAgent` 是 multi-step loop，循环退出条件 `IsDone(thread)` 完全靠 thread 推导。

## Solution / 解决方案

3 个决策：

1. **`RunAgent` 是纯函数式 loop（除了 thread.Append）**：进去一个 Thread，出来同一个 Thread（mutated），加一个 error。不接 CLI，不接 HTTP。让 s06 的 server 和 s10 的 orchestrator 都能复用。
2. **每个动作都进 thread**：tool_call 和 tool_response 一对一。done_for_now 也作为 tool_call 进 thread —— 这样 thread 是一份完整决策记录。
3. **`MaxSteps` 兜底**：写死 16。生产代码会用 `context.WithTimeout`，但教学场景用 const 更显式（看一眼代码就懂"loop 最多跑多少"）。

## How It Works / 工作原理

```
   NewThread(user_input) ──► RunAgent ──┐
                                         │
                              ┌──────────┴────────┐
                              │ for step < Max:    │
                              │   provider call   │ ◄──── thread.SerializeForLLM()
                              │   append tool_call│
                              │   if done: return │
                              │   Registry lookup │
                              │   tool.Execute    │
                              │   append response │
                              └─────────┬─────────┘
                                        │
                                        ▼
                                  final Thread
```

核心 30 行（节选自 [`agents/s05-unify-execution-state/loop.go`](https://github.com/Ding-Ye/learn-12-factor-agents/blob/main/agents/s05-unify-execution-state/loop.go)）：

```go
func RunAgent(ctx context.Context, thread *Thread, provider Provider, registry Registry) (*Thread, error) {
    for step := 0; step < MaxSteps; step++ {
        next, err := provider.DetermineNextStep(ctx, thread.SerializeForLLM())
        if err != nil { return thread, fmt.Errorf("provider call at step %d: %w", step, err) }

        thread.Append(NewToolCallEvent(next))
        if next.Intent == IntentDoneForNow { return thread, nil }

        tool, err := registry.Lookup(next.Intent)
        if err != nil { return thread, fmt.Errorf("dispatch at step %d: %w", step, err) }

        result, err := tool.Execute(ctx, next.Data)
        if err != nil { return thread, fmt.Errorf("tool %q at step %d: %w", next.Intent, step, err) }

        thread.Append(NewToolResponseEvent(result))
    }
    return thread, fmt.Errorf("agent loop hit MaxSteps=%d", MaxSteps)
}
```

**3 个非显然之处**：

1. **done_for_now 也进 thread**：把"我决定结束"当成一个 tool_call 记录。这样 thread 的最后一条永远是 tool_call，便于 `IsDone(thread)` 一行实现（"最后一条 tool_call 的 intent 是不是 done"）。s12 的 replay 测试也依赖这条记录。
2. **Provider 在每次 loop 都接一份新序列化的 thread**：不是 incremental。每轮都重新看完整的历史。token 成本高，但匹配真实 LLM API 的无状态特性，也让 ScriptedSequenceProvider 完全不需要状态机制。
3. **`MaxSteps = 16` 不是任意数**：上游 factor-10 说"3-20 步是小 agent 的合理上限"。16 留 25% buffer 给意外的退避循环。s09 的错误计数会用同一个上限。

## What Changed / 与 s04 的变化

```diff
- main.go: single dispatch
+ loop.go (新建 — RunAgent)
+ thread.go: + IsDone + LastToolCall (基于 thread 推导执行状态)
+ provider.go: ScriptedSequenceProvider (多步 canned)
  tools.go: unchanged (4 math tools + Registry)
  events.go: NewToolCallEvent 现在接 NextStep (而不是 generic any)
- 7 tests
+ 6 tests (loop_test.go - 多步 + IsDone + Last + 错误传播)
```

语义上的差别：s04 是"一次 LLM 调用 + 一次 tool 执行"；s05 是"循环直到 done"。Thread 第一次承载多轮 tool_call/tool_response。

## Try It / 动手试一试

```bash
cd agents/s05-unify-execution-state

go test -v ./...

go run .
# Loop ran 3 turns. Final thread has 6 events.
#   [0] user_input: "add 5 and 3, then multiply by 2"
#   [1] tool_call: intent=add
#   [2] tool_response: 8
#   [3] tool_call: intent=multiply
#   [4] tool_response: 16
#   [5] tool_call: intent=done_for_now
```

3 步：add(5,3)=8 → multiply(8,2)=16 → done。6 个 events：1 user_input + 3 tool_call + 2 tool_response（done_for_now 没有 tool_response）。

## Upstream Source Reading / 上游源码阅读

```upstream:workshops/2025-07-16/walkthrough/03-agent.py#L14-L37
# Source: workshops/2025-07-16/walkthrough/03-agent.py lines 14-37
# License: Apache 2.0

def agent_loop(thread: Thread) -> AgentResponse:
    b = get_baml_client()
    while True:
        next_step = b.DetermineNextStep(thread.serialize_for_llm())
        thread.events.append({
            "type": next_step.intent,
            "data": next_step
        })
        
        if next_step.intent == "done_for_now":
            return next_step
        elif next_step.intent == "add":
            result = next_step.a + next_step.b
            thread.events.append({
                "type": "tool_response",
                "data": result
            })
        elif next_step.intent == "subtract":
            # ... etc
            pass
```

**对照阅读要点**：

- **`while True` vs `for step < MaxSteps`**：上游没兜底，我们加 MaxSteps。生产场景必须有兜底（context.WithTimeout / 步数）。
- **per-intent dispatch in switch** vs **Registry lookup**：上游用 if/elif 链；我们 Registry map。功能等价，Registry 更易扩展（s10 sub-agents 复用）。
- **`thread.events.append({"type": next_step.intent, "data": next_step})`**：上游 type 用 intent 字符串（"add"），我们 type 用 const "tool_call"（更通用）。"intent 是什么"读 data.Intent 拿。
- **没有 IsDone helper**：上游通过 intent 字符串 inline 比较；我们抽 helper。两种都对，helper 让 s06 server / s12 reducer 复用同一个判定。
- **没有 LastToolCall helper**：上游不需要（loop 内变量直接保有）；我们暴露让外部代码（s06 HTTP handler）能问"上一步做了啥"。

**想读更多**：`content/factor-05-unify-execution-state.md` 全文 + `workshops/2025-07-16/walkthrough/03-agent.py` 全文。配合看你能感觉"为什么 unified 比 split 简单"。

---

**下一节预告**：s06 把 RunAgent 从命令行解放出来 —— 引入 `net/http` server + ThreadStore。`POST /thread` 启动一个 thread，`GET /thread/{id}` 看状态，`POST /thread/{id}/response` 续传。
