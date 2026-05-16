---
title: "s10 · Small, focused agents"
chapter: 10
slug: s10-small-focused-agents
est_read_min: 8
---

# s10 · Small, focused agents

> What you learn here: instead of one agent burning 20+ steps and a giant context, split the work across small sub-agents. `Orchestrator` composes `CalcAgent` and `SummaryAgent`; each owns its own thread, context, and tests.

---

## Problem / The gap

s09's loop can self-heal and step through complex flows, but it's still **one** big agent. Real tasks expose two failure modes:

- 30+ events in the thread, prompts overflow context, tokens get expensive
- one agent has to handle both "do math" and "write the user-facing summary"; the prompts for those skills interfere
- a 30-step agent is virtually untestable in isolation

Upstream factor-10: **3-20 steps is the upper bound for "small."** Bigger work uses an orchestrator to compose sub-agents.

## Solution / Mental model

Three decisions:

1. **Sub-agents live in a sub-package**: `subagents/calc.go` and `subagents/summary.go`. They don't know `Orchestrator` exists — `CalcAgent(in CalcInput) (CalcOutput, error)` is a pure function.
2. **The orchestrator's thread records only boundaries**: `subagent_call` and `subagent_done`. How many steps CalcAgent took internally, and on which intermediate values, never enters the orchestrator's context. That's literally what "small context" means.
3. **Every sub-agent has typed I/O structs**: `CalcInput{Steps []CalcStep}` / `SummaryInput{Result float64; UserMessage string}`. They're individually unit-testable.

## How It Works

```
   user_input ──► Orchestrator
                       │
                       ├── append subagent_call(CalcAgent, plan)
                       │   CalcAgent: add → multiply → result=16
                       └── append subagent_done(CalcAgent, {result:16})
                       │
                       ├── append subagent_call(SummaryAgent, {result:16, msg:...})
                       │   SummaryAgent: "Computed 16 for: add 5 and 3..."
                       └── append subagent_done(SummaryAgent, final)
                       │
                       └─► final user-facing string
```

Core 30 lines (excerpt from `orchestrator.go`):

```go
func Orchestrate(thread *Thread, userMessage string, plan subagents.CalcInput) (string, error) {
    thread.Append(NewSubAgentCallEvent("CalcAgent", plan))
    calcOut, err := subagents.CalcAgent(plan)
    if err != nil {
        return "", fmt.Errorf("CalcAgent: %w", err)
    }
    thread.Append(NewSubAgentDoneEvent("CalcAgent", calcOut))

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

**Three non-obvious bits:**

1. **`subagents` is a sub-package**, not another module — sub-agents are implementation details of the orchestrator and shouldn't cross module boundaries.
2. **`CalcAgent` doesn't take `ctx`** — it's pure. Sub-agents that talk to an LLM (Phase G work) would need ctx.
3. **`subagent_done` wraps the entire `CalcOutput`** (result + trace). The orchestrator's thread doesn't unfold the trace into individual `tool_call` / `tool_response` events — that's exactly the context-compression the factor demands.

## What Changed vs s09

```diff
+ subagents/ sub-package
+   - calc.go: CalcAgent + CalcInput/CalcStep/CalcOutput
+   - summary.go: SummaryAgent + SummaryInput
+ orchestrator.go: Orchestrate
+ events.go: + EventTypeSubAgentCall / SubAgentDone
- 6 tests
+ 6 tests (sub-agent isolation + no low-level tool events in orchestrator thread)
```

Semantically: s09 = one thread with N steps; s10 = an orchestrator thread with K sub-agent calls, each sub-agent owning its M internal steps. Context budget drops from O(N) to O(K), with K ≪ N.

## Try It

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

Notice the thread contains **no** `tool_call` / `tool_response` — CalcAgent ran two math steps internally, but the orchestrator only sees one call/done pair.

## Upstream Source Reading

```upstream:content/factor-10-small-focused-agents.md#L1-L41
# Source: content/factor-10-small-focused-agents.md lines 1-41
# License: CC BY-SA 4.0

# Factor 10: Small, focused agents

Most successful agents in production keep individual agent loops between
3-20 steps. Beyond that, two failure modes show up:

1. Context bloat: Each step adds tokens to the prompt. Past 20 steps,
   the LLM starts losing earlier context, hallucinating, or just running
   out of context window.

2. Mixed concerns: A 50-step agent often handles many different
   sub-tasks. The prompts for those sub-tasks are different. Sharing
   one prompt is a recipe for confusion.

The fix: compose small agents. Each one does one thing. The outer
orchestrator (just normal code!) decides which agent runs when, and
threads the data between them.

This is the inverse of "let the LLM plan everything." We're saying: when
the structure of the work is known in advance (most production cases),
write the structure as code.
```

**Reading notes:**

- **Upstream is markdown-only, no reference impl**: factor-10's `.md` describes a pattern; concrete implementations are up to you. We ship one minimal demo (CalcAgent + SummaryAgent).
- **"Orchestrator is normal code"**: upstream stresses the orchestrator doesn't need to be an LLM — if/else, state machine, DAG all qualify. Our `Orchestrate` is a sync Go function and fits the bill.
- **3-20 steps per sub-agent**: CalcAgent runs 2 (add + multiply); SummaryAgent runs 1 (template). Both safely below the ceiling.
- **What we omit**: data-flow problems between sub-agents (e.g., CalcAgent output shape mismatch → SummaryAgent fallback). Production uses retry / fallback orchestrators.

**Want to read more?** `content/factor-10-small-focused-agents.md` plus `content/brief-history-of-software.md` (why "small is beautiful" still applies in the agent era).

---

**Up next, s11:** CLI is no longer the only launcher. A `Trigger` interface ingests Slack webhooks, HumanLayer events, and other external sources — translating them into threads before reusing s06's `RunAgent`.
