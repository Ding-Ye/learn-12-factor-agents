---
title: "s10 · 小而专一的子 agent 编排"
chapter: 10
slug: s10-small-focused-agents
est_read_min: 8
---

# s10 · 小而专一的子 agent 编排

> 教什么：与其一个 agent 跑 20+ 步把上下文撑爆，不如让 Orchestrator 把任务拆给 CalcAgent / SummaryAgent 各跑 3-5 步。每个 sub-agent 独立 thread，独立 context，独立测试。

---

## Problem / 问题

s09 让单 agent 能 self-heal、多步推进，但仍是**一个**大 agent。真实场景里 task 复杂后会出现：

- thread 里塞了 30+ 个 event；prompt 装不下，token 又贵
- 一个 agent 同时要懂"算数学"和"用人类语气总结"；这两件事的 prompt 互相干扰
- 测试一个跨 30 步的 agent 几乎不可能 isolate

上游 factor-10：**3-20 步是合理的 agent 大小上限**。任务大了就用 orchestrator 拆 sub-agent。

## Solution / 解决方案

3 个决策：

1. **Sub-agent 是包内子包**：`subagents/calc.go` 和 `subagents/summary.go`。它们不知道 orchestrator 存在 —— `CalcAgent(in CalcInput) (CalcOutput, error)` 是纯函数。
2. **Orchestrator thread 只记 boundary**：`subagent_call` 和 `subagent_done`。CalcAgent 内部跑了多少步、用了什么中间值，orchestrator 都不看。这就是"上下文小"的精确含义。
3. **每个 sub-agent 输入/输出都是 typed struct**：`CalcInput{Steps []CalcStep}` / `SummaryInput{Result float64; UserMessage string}`。可以单独 unit-test。

## How It Works / 工作原理

```
   user_input ──► Orchestrator
                       │
                       ├── append subagent_call(CalcAgent, plan)
                       │   CalcAgent runs: add → multiply → result=16
                       └── append subagent_done(CalcAgent, {result:16})
                       │
                       ├── append subagent_call(SummaryAgent, {result:16, msg:...})
                       │   SummaryAgent: "Computed 16 for: add 5 and 3..."
                       └── append subagent_done(SummaryAgent, final)
                       │
                       └─► final user-facing string
```

核心 30 行（节选自 [`agents/s10-small-focused-agents/orchestrator.go`](https://github.com/Ding-Ye/learn-12-factor-agents/blob/main/agents/s10-small-focused-agents/orchestrator.go)）：

```go
func Orchestrate(thread *Thread, userMessage string, plan subagents.CalcInput) (string, error) {
    // Delegate to CalcAgent
    thread.Append(NewSubAgentCallEvent("CalcAgent", plan))
    calcOut, err := subagents.CalcAgent(plan)
    if err != nil {
        return "", fmt.Errorf("CalcAgent: %w", err)
    }
    thread.Append(NewSubAgentDoneEvent("CalcAgent", calcOut))

    // Delegate to SummaryAgent
    thread.Append(NewSubAgentCallEvent("SummaryAgent", subagents.SummaryInput{
        UserMessage: userMessage,
        Result:      calcOut.Result,
    }))
    final := subagents.SummaryAgent(subagents.SummaryInput{
        UserMessage: userMessage,
        Result:      calcOut.Result,
    })
    thread.Append(NewSubAgentDoneEvent("SummaryAgent", final))

    return final, nil
}
```

**3 个非显然之处**：

1. **`subagents` 是子包**，不是另一个 module：sub-agent 是 orchestrator 的实现细节，不该跨 module 边界。
2. **`CalcAgent` 是纯函数（不接 ctx）**：因为它不需要 IO 也不要超时。如果未来接 LLM 才需要 `ctx`。
3. **`subagent_done` 把整个 `calcOut` 塞进 event Data**：包括 `Trace` 字段（调试用）。orchestrator thread 里不展开成多个 `tool_call/tool_response` —— 这就是"压缩上下文"。

## What Changed / 与 s09 的变化

```diff
+ subagents/ (子包)
+   - calc.go: CalcAgent + CalcInput/CalcStep/CalcOutput
+   - summary.go: SummaryAgent + SummaryInput
+ orchestrator.go: Orchestrate function
+ events.go: + EventTypeSubAgentCall / SubAgentDone
- 6 tests
+ 6 tests (含子 agent 独立性 + thread 不含 low-level tool 调用)
```

语义上的差别：s09 = 一个 thread 装 N 步；s10 = 一个 orchestrator thread 装 K 个 sub-agent 调用，每个 sub-agent 自己内部跑 M 步。Context budget 从 O(N) 降到 O(K) （K << N）。

## Try It / 动手试一试

```bash
cd agents/s10-small-focused-agents

go test -v ./...

go run .
# Computed 16 for: add 5 and 3, then multiply by 2
# 
# Orchestrator thread:
#   [0] user_input
#   [1] subagent_call
#   [2] subagent_done
#   [3] subagent_call
#   [4] subagent_done
```

注意 thread 里**没有** `tool_call` / `tool_response`。CalcAgent 内部跑了 2 步 add+multiply，但 orchestrator 只看到一对 call/done。

## Upstream Source Reading / 上游源码阅读

```upstream:content/factor-10-small-focused-agents.md#L1-L41
# Source: content/factor-10-small-focused-agents.md lines 1-41
# License: CC BY-SA 4.0

# Factor 10: Small, focused agents

Most successful agents in production keep individual agent loops between
3-20 steps. Beyond that, two failure modes show up:

1. **Context bloat**: Each step adds tokens to the prompt. Past 20 steps,
   the LLM starts losing earlier context, hallucinating, or just running
   out of context window.

2. **Mixed concerns**: A 50-step agent often handles many different
   sub-tasks. The prompts for those sub-tasks are different. Sharing
   one prompt is a recipe for confusion.

The fix: compose small agents. Each one does one thing. The outer
orchestrator (just normal code!) decides which agent runs when, and
threads the data between them.

This is the inverse of "let the LLM plan everything." We're saying: when
the structure of the work is known in advance (most production cases),
write the structure as code.
```

**对照阅读要点**：

- **上游只有 markdown，没有 reference 实现**：upstream `content/factor-10-*.md` 写的是 pattern；具体怎么写代码，每家自由发挥。我们 Go 端写了一个最小 demo（CalcAgent + SummaryAgent）。
- **"orchestrator 是普通代码" vs "另一个 agent"**：上游强调 orchestrator 不需要是 LLM；可以是 if/else、状态机、DAG。我们的 `Orchestrate` 是 sync Go 函数，符合这条原则。
- **每个 sub-agent 应该 3-20 步**：CalcAgent 跑 2 步（add + multiply），SummaryAgent 跑 1 步（template）。比上限远小，正是 factor 鼓励的。
- **缺的部分**：上游隐含的"sub-agent 之间数据流"问题（如 CalcAgent 输出 dim mismatch 时 SummaryAgent 怎么 fallback）我们没做。生产中通常用 retry / fallback orchestrator。

**想读更多**：`content/factor-10-small-focused-agents.md` 整篇 + `content/brief-history-of-software.md`（讲为什么"小是美的"传统在 agent 时代依然适用）。

---

**下一节预告**：s11 让 agent 不再只能从 CLI 启动 —— `Trigger` 接口接 Slack webhook、HumanLayer event 等任意外部源，转换成 thread 后复用 s06 的 `RunAgent`。
