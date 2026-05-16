---
title: "s12 · 无状态 reducer (replay + fork)"
chapter: 12
slug: s12-stateless-reducer
est_read_min: 9
---

# s12 · 无状态 reducer (replay + fork)

> 教什么：把整个 agent loop 重构成纯函数 `Reduce(Thread, Event) Thread`。无 side effect 无全局态。Replay 测试一行，Fork 测试一行。

---

## Problem / 问题

s05-s11 的 `RunAgent` 一直用 `*Thread` 指针。优点是 append O(1)；缺点是：

- 测试很难写"同一 thread 跑两次得到一样的结果" —— 因为 thread 被 mutate 了
- 想 fork 一个 thread 到两条分支，得自己写 deep copy
- 没法把"步骤"独立 isolation —— 谁知道哪一行 mutate 了 thread

上游 factor-12 的论点：**agent 就是一个 reducer**。`f(state, event) = next_state`。和 Redux 一样，本质上是 fold over events。

## Solution / 解决方案

3 个决策：

1. **`Reduce(Thread, Event) Thread` 接 value，返 value**：不接 pointer 也不返 error。reducer 内部失败（如 unmarshal）就 append `error` event 到返回的 Thread 上 —— error 也是数据。
2. **`Thread.Append` 永远 copy 新 slice**：`make([]Event, len+1)` + `copy`。开销 O(n)，但保证 fork 不互相干扰。生产场景可以 immutable.js-style 路径压缩，教学场景显式 copy 更清楚。
3. **Reduce 在 tool_call 后 auto-step append tool_response**：让单次 Reduce 调用代表一个完整的"决策 + 执行"周期。这样 replay 测试不用模拟"两个相邻 event"——一个 event 一次 settle。

## How It Works / 工作原理

```
   Thread{}  ──► Reduce(t, user_input)
              ──► Reduce(t, tool_call(add 5,3))
                    │
                    └── auto-step: 算 5+3=8, append tool_response(8)
              ──► Reduce(t, tool_call(multiply 8,2))
                    │
                    └── auto-step: 算 8*2=16, append tool_response(16)
              ──► Reduce(t, tool_call(done_for_now))
                    │
                    └── (no auto-step; loop terminates)
              ──► IsDone(t) == true
```

核心 30 行（节选自 [`reducer.go`](https://github.com/Ding-Ye/learn-12-factor-agents/blob/main/agents/s12-stateless-reducer/reducer.go)）：

```go
func Reduce(t Thread, e Event) Thread {
    t = t.Append(e)

    if e.Type != EventTypeToolCall {
        return t
    }
    var step NextStep
    if err := json.Unmarshal(e.Data, &step); err != nil {
        return t.Append(NewEvent("error", fmt.Sprintf("decode tool_call: %v", err)))
    }
    switch step.Intent {
    case IntentAdd:
        var p MathPayload
        _ = json.Unmarshal(step.Data, &p)
        return t.Append(NewEvent(EventTypeToolResponse, p.A+p.B))
    case IntentMultiply:
        var p MathPayload
        _ = json.Unmarshal(step.Data, &p)
        return t.Append(NewEvent(EventTypeToolResponse, p.A*p.B))
    case IntentDoneForNow:
        return t  // terminal
    default:
        return t.Append(NewEvent("error", fmt.Sprintf("unknown intent %q", step.Intent)))
    }
}

func ReduceMany(t Thread, events []Event) Thread {
    for _, e := range events { t = Reduce(t, e) }
    return t
}
```

**3 个非显然之处**：

1. **Thread 是 value type**：copy 一份是 cheap operation（slice header 是 24 字节）。`func (t Thread) Append` 返回新的 Thread；老的不变。这就是"immutable-by-convention"。
2. **`Reduce` 不接 ctx / 不接 provider**：纯函数。tool 执行硬编码在 reducer 里。这是教学版的简化 —— 生产代码会把 tool 抽出来，但纯函数原则不变（外部 IO 留给 caller，reducer 内只算）。
3. **`Equal` 比较走 JSON marshal 不走 `reflect.DeepEqual`**：`json.RawMessage` 是 `[]byte` —— `DeepEqual` 在 nil vs empty slice 上行为有 bug。JSON marshal 抽象出"是否字节相等"。

## What Changed / 与 s11 的变化

```diff
+ reducer.go: Reduce + ReduceMany + IsDone (s12 主角)
+ thread.go: Thread 改 value type，Append 返新 Thread，加 Equal
- events.go: Data 改 json.RawMessage（替 interface{}）
- loop.go: RunAgent 改成 Reduce 的薄壳
- 5 tests
+ 6 tests (含 replay + fork + no-mutation + auto-step)
```

语义上的差别：s11 以前每章都是"loop body mutates thread"；s12 改成"loop body computes the next thread"。看上去差别小，但是 testability、replayability、forkability 全面升级。

## Try It / 动手试一试

```bash
cd agents/s12-stateless-reducer

go test -v ./...
# 6 PASS：含 replay、fork、no-mutation、auto-step、end-to-end

go run .
# Final thread (6 events):
#   [0] user_input: "add 5 and 3, then multiply by 2"
#   [1] tool_call: {"intent":"add","data":{"a":5,"b":3}}
#   [2] tool_response: 8
#   [3] tool_call: {"intent":"multiply","data":{"a":8,"b":2}}
#   [4] tool_response: 16
#   [5] tool_call: {"intent":"done_for_now","data":{"message":"Result is 16."}}
```

注意 Replay 测试只有 3 行：

```go
a := ReduceMany(Thread{}, events)
b := ReduceMany(Thread{}, events)
if !a.Equal(b) { t.Fatal("replay failed") }
```

如果 Reduce 不是纯函数，这个测试无法这么短地成立。

## Upstream Source Reading / 上游源码阅读

```upstream:packages/create-12-factor-agent/template/src/agent.ts#L89-L114
// Source: packages/create-12-factor-agent/template/src/agent.ts lines 89-114
// License: Apache 2.0

export async function agentLoop(thread: Thread): Promise<Thread> {
    while (true) {
        const nextStep = await b.DetermineNextStep(thread.serializeForLLM());
        
        thread.events.push({
            "type": "tool_call",
            "data": nextStep
        });
        
        if (nextStep.intent === "done_for_now") {
            return thread;
        }
        
        // Execute tool — TypeScript switch on intent
        if (nextStep.intent === "add") {
            const result = nextStep.a + nextStep.b;
            thread.events.push({"type": "tool_response", "data": result});
        }
        // ... more tools
    }
}
```

```upstream:content/factor-12-stateless-reducer.md#L1-L12
# Source: content/factor-12-stateless-reducer.md lines 1-12
# License: CC BY-SA 4.0

# Factor 12: Make your agent a stateless reducer

Functions all the way down. Your agent is a thread of events. Each
event is processed by a pure function that returns the next thread.
No mutable state outside the thread. No global variables. Just data
flowing through transformations.

Replay becomes trivial — you can re-run any subsequence of events. Fork
becomes trivial — you can branch at any point. Test becomes trivial —
you can assert exact outputs for exact inputs.
```

**对照阅读要点**：

- **上游 `agentLoop` 还在 mutate**：`thread.events.push(...)` 是直接改 array。markdown 里讲的 reducer 是 aspirational target，参考实现没完全落地。我们 Go 端真正实现了纯 reducer。
- **`thread.events.push` 在 TypeScript 是 O(1) 平均**：但破坏 immutability。我们 Go 用 copy，O(n) 但 immutable。生产场景可以用 persistent vector（如 `github.com/benbjohnson/immutable`）解决这个 trade-off。
- **上游 markdown 强调 "functions all the way down"**：我们的 Reduce + ReduceMany + IsDone 都是纯函数，符合这个口号。
- **上游 `b.DetermineNextStep` 在 loop 内 await**：IO 在 loop 体；我们 Go 把 IO 移到 caller（loop.go 的 RunAgent），Reduce 自身不接 IO。这是教学版的额外收益。
- **缺的部分**：上游强调 replay+fork 但 reference impl 不能直接做（mutable）。我们 Go 端的 replay+fork 测试是真能跑的。

**想读更多**：`content/factor-12-stateless-reducer.md` 全篇（12 行）+ `packages/create-12-factor-agent/template/src/agent.ts:89-114`。配合 React/Redux 的 reducer 文档读，能感觉"agent 就是 LLM 驱动的 reducer"这个 mental model 的力量。

---

## 整本课程结束

s01 → s12 走完了。`s_full` 集成章节、附录 A（Agents are mostly software）、附录 B（上游源码导读地图）在仓库根的 `docs/` 里。
