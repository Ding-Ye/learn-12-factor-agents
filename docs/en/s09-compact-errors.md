---
title: "s09 · Compact errors into context"
chapter: 9
slug: s09-compact-errors
est_read_min: 8
---

# s09 · Compact errors into context

> What you learn here: tool failures stop terminating the agent. They become `error` events on the thread, the LLM reads them next turn and self-corrects. Three consecutive errors escalate.

---

## Problem / The gap

s08's `RunAgent` exited the moment a tool returned an error. But in real flows:

- the model emits `divide(10, 0)` → tool yields "Division by zero" → exiting now means the agent never tries an alternative
- on the next turn, having seen the error, the model can pick `divide(10, 2)` and recover

Upstream factor-09's argument: **errors are data**. Stuff them into the thread as events for the LLM to consume; only escalate when the model keeps making the same mistake (consecutive N).

## Solution / Mental model

Three decisions:

1. **`NewErrorEvent` is a new event type** — appended right after a `tool_call`, with no `tool_response`. The LLM seeing the `(tool_call, error)` pair has all it needs to retry.
2. **`ConsecutiveErrors(thread)` walks the tail** — counts `(tool_call, error)` pairs walking backwards; the first non-error event ends the streak.
3. **`SafeExecute` recovers panics** — a buggy tool shouldn't crash the agent. Panics turn into errors and follow the same self-heal path.

## How It Works

```
   RunAgent step N:
       provider → next (e.g., divide(10, 0))
       append tool_call(next)
       tool.Execute → error "Division by zero"
       ┌──────────────────────────────────────────┐
       │ append NewErrorEvent(err.Error())        │
       │ if ConsecutiveErrors(thread) >= 3 → escalate │
       │ else: continue                           │
       └──────────────────────────────────────────┘

   step N+1:
       provider sees thread including the error event
       returns NextStep{intent: "divide", a: 10, b: 2}   ← LLM self-corrected
       tool.Execute → 5  (success!)
       append tool_response(5)
```

Core 30 lines (excerpts from `thread.go` + `loop.go`):

```go
func ConsecutiveErrors(t *Thread) int {
    count := 0
    for i := len(t.Events) - 1; i >= 0; i-- {
        switch t.Events[i].Type {
        case EventTypeError:
            count++
        case EventTypeToolCall:
            continue   // paired tool_call doesn't break the streak
        default:
            return count   // any other event ends the streak
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

**Three non-obvious bits:**

1. **`ConsecutiveErrors` walks tail-first** — only counts the recent streak. A success in the middle resets it.
2. **`SafeExecute` uses named returns** — `defer func() { ... }()` can only write `err` if it's a named return variable. This is Go's idiomatic panic-recover pattern.
3. **Escalation is an explicit error return** — not a panic, not a silent break. The HTTP handler sees the error and can decide what status code to use, what to log, whether to page someone.

## What Changed vs s08

```diff
+ types.go: + MaxConsecutiveErrors = 3 + IntentDivide
+ thread.go: + ConsecutiveErrors helper
+ tools.go: + DivideTool, SafeExecute (panic recover)
- loop.go: tool error returns immediately
+ loop.go: append error event, check counter, continue or escalate
- 7 tests
+ 6 tests (including panic + escalation paths)
```

Semantically: s08's failure = termination; s09's failure = opportunity. The LLM gains the ability to "see the error and try again."

## Try It

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

The first `divide(10, 0)` errors and becomes an event; the second `divide(10, 2)` succeeds with 5; then done. The entire trace records the error, the recovery, and the finish.

## Upstream Source Reading

```upstream:workshops/2025-07-16/walkthrough/03-agent.py#L21-L35
# Source: workshops/2025-07-16/walkthrough/03-agent.py lines 21-35
# License: Apache 2.0

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

**Reading notes:**

- **`try/except` vs `SafeExecute + recover`** — upstream's `except Exception` catches everything; we split panic from error and converge them in `SafeExecute`.
- **Loop-local `consecutive_errors` vs `ConsecutiveErrors(thread)`** — upstream stores the counter in a loop variable; we derive it from the thread. Our advantage: HTTP resume reconstructs the count from the persisted thread for free.
- **`format_error(e)` vs `err.Error()`** — upstream uses a separate function to strip the Python traceback; Go's `err.Error()` is already compact.
- **Upstream `break` vs our explicit return** — escalation in upstream just exits the loop with no error; we return one. Trade-off depends on who notices "the agent got stuck."
- **What upstream's markdown adds** — `content/factor-09-compact-errors.md` argues for compacting tracebacks (drop file paths, call stacks). Our `err.Error()` is already concise, but production code might add an extra summarization step.

**Want to read more?** `content/factor-09-compact-errors.md:10-59` is the canonical "why not fast-fail" essay.

---

**Up next, s10:** sub-agent orchestration. An `Orchestrator` splits tasks across `CalcAgent` (math) and `SummaryAgent` (natural-language wrap-up). Each sub-agent has its own thread — small contexts, independently testable, parallelisable.
