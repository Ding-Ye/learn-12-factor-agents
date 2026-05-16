# Upstream reading — Factor 9: Compact errors into context

> Source: humanlayer/12-factor-agents @ d20c728.

## Source: `workshops/2025-07-16/walkthrough/03-agent.py` (21-35)

```python
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

## Mapping

| Upstream | Our Go | File |
|---|---|---|
| `try/except` | `SafeExecute` + `recover()` | tools.go |
| Loop-local `consecutive_errors` | `ConsecutiveErrors(thread)` derived | thread.go |
| `format_error(e)` | plain `err.Error()` | loop.go |
| `break` to escalate | `return thread, fmt.Errorf(...)` | loop.go |

## Why derive the counter from the thread

Upstream's loop-local counter is fine for a single CLI invocation. Ours
must survive HTTP request boundaries — the agent runs in step N during
one POST, then the handler returns, the thread is persisted, and the
NEXT POST may resume execution. Storing the counter in the loop would
lose it on resume.

By deriving it from `thread.Events`, we get the right answer
automatically wherever the thread is loaded from.

## Quoted prose

From `content/factor-09-compact-errors.md` lines 10-20 (CC-BY-SA 4.0):

> When a tool call fails, the natural reaction is to surface the
> exception to the caller. But agents are different: failures should be
> opportunities for the model to self-correct. Wrap each tool call,
> capture the error, append it to the thread as a structured event, and
> let the next LLM call decide what to do.

## Reading map

- s09 → s10: sub-agents inherit the same compact-error pattern.
- s09 → s12: the `Reduce(Thread, Event) -> Thread` shape makes error
  events first-class; replay tests reproduce the same self-heal path.
