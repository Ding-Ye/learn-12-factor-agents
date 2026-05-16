---
title: "s05 · Unify execution state"
chapter: 5
slug: s05-unify-execution-state
est_read_min: 9
---

# s05 · Unify execution state

> What you learn here: every "execution-time" fact (which step we're on, how many errors in a row, who we're waiting for) lives in `Thread.Events` — no parallel state.

---

## Problem / The gap

s04 nailed single-step dispatch. Real agents loop until done. The common anti-pattern is to introduce an "execution state" record (current_step, is_waiting, error_count) **alongside** the business state (user_input, tool_results). The two records drift out of sync — business state says step 3 finished, execution state still says step 2.

Upstream factor-05's answer: **keep one state — `Thread.Events`**. Every derived fact is a query over that list. s05 lands this in Go: `RunAgent` is a multi-step loop and its exit condition `IsDone(thread)` is derived purely from the thread.

## Solution / Mental model

Three decisions:

1. **`RunAgent` is a (nearly) pure loop**: takes a Thread, returns the same (mutated) Thread + error. No CLI, no HTTP. s06's server and s10's orchestrator both call this same function.
2. **Every action enters the thread**: tool_call and tool_response are paired. Even `done_for_now` is a tool_call — that way the thread is a complete decision record.
3. **`MaxSteps` as a backstop**: hard-coded to 16. Production would use `context.WithTimeout`; we keep a const for teaching clarity.

## How It Works

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

Core 30 lines (excerpt from [`agents/s05-unify-execution-state/loop.go`](https://github.com/Ding-Ye/learn-12-factor-agents/blob/main/agents/s05-unify-execution-state/loop.go)):

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

**Three non-obvious bits:**

1. **`done_for_now` also goes into the thread** as a tool_call. That keeps the loop-exit predicate `IsDone(thread)` to one line ("last tool_call's intent is done"). s12's replay test depends on this.
2. **The provider receives a freshly serialized thread every iteration** — not incremental. Each turn it sees the full history. Costs tokens, but matches real LLM APIs (which are stateless) and lets `ScriptedSequenceProvider` ignore state.
3. **`MaxSteps = 16` isn't arbitrary** — upstream factor-10 calls 3–20 steps the "small agent" range. 16 leaves headroom; s09's error counter uses the same ceiling.

## What Changed vs s04

```diff
- main.go: single dispatch
+ loop.go (new — RunAgent)
+ thread.go: + IsDone + LastToolCall (derived execution state)
+ provider.go: ScriptedSequenceProvider (multi-step canned)
  tools.go: unchanged (4 math tools + Registry)
  events.go: NewToolCallEvent now takes a NextStep (not generic any)
- 7 tests
+ 6 tests (loop_test.go — multi-step + IsDone + Last + error paths)
```

Semantically: s04 was "one LLM call + one tool execution"; s05 is "loop until done." The Thread carries multi-turn tool_call/tool_response pairs for the first time.

## Try It

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

Three turns: add(5,3)=8 → multiply(8,2)=16 → done. Six events: 1 user_input + 3 tool_call + 2 tool_response (done_for_now has no tool_response).

## Upstream Source Reading

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

**Reading notes:**

- **`while True` vs `for step < MaxSteps`**: upstream has no cap; we add one. Production needs a cap (`context.WithTimeout` or step count).
- **Per-intent if/elif** vs **Registry lookup**: upstream chains conditionals; we use a Registry map. Functionally equivalent, but the Registry composes (s10 sub-agents reuse it).
- **`thread.events.append({"type": next_step.intent, ...})`**: upstream uses the intent string as event type (`"add"`); we use a const `"tool_call"` and read `data.Intent` for the intent.
- **No `IsDone` helper**: upstream inlines the string compare; we extract a helper so s06's HTTP handler and s12's reducer can share it.
- **No `LastToolCall` helper**: upstream doesn't need it (loop-local variables already hold the value); we expose it so external code (s06's `/thread/{id}` GET) can ask "what did the agent just do?"

**Want to read more?** `content/factor-05-unify-execution-state.md` plus the full `03-agent.py`. The two together make clear why one source of truth is simpler than two.

---

**Up next, s06:** liberate `RunAgent` from the CLI. Introduce a `net/http` server + `ThreadStore`. `POST /thread` starts; `GET /thread/{id}` reads; `POST /thread/{id}/response` resumes.
