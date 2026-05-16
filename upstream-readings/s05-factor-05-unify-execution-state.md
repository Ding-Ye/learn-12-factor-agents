# Upstream reading — Factor 5: Unify execution state

> Source: humanlayer/12-factor-agents @ d20c728.

## Source: `workshops/2025-07-16/walkthrough/03-agent.py` (lines 14-37)

```python
# License: Apache 2.0
def agent_loop(thread: Thread) -> AgentResponse:
    b = get_baml_client()
    while True:
        next_step = b.DetermineNextStep(thread.serialize_for_llm())
        thread.events.append({"type": next_step.intent, "data": next_step})

        if next_step.intent == "done_for_now":
            return next_step
        elif next_step.intent == "add":
            result = next_step.a + next_step.b
            thread.events.append({"type": "tool_response", "data": result})
        elif next_step.intent == "subtract":
            ...
```

## Mapping table

| Upstream concept | Our Go equivalent | File |
|---|---|---|
| `while True` | `for step < MaxSteps` (16) | `loop.go` |
| `if/elif` per intent | `Registry.Lookup` + `Tool.Execute` | `tools.go`, `loop.go` |
| Inline `next_step.intent == "done_for_now"` | `IsDone(thread)` helper | `thread.go` |
| Implicit "what did the agent just do?" | `LastToolCall(thread)` | `thread.go` |
| `agent_loop` returns AgentResponse | `RunAgent` returns `*Thread, error` | `loop.go` |

## Why we make the loop bail at MaxSteps

A `while True` is fine in a notebook (you Ctrl-C); it's not fine in a
service. Hard-coding `MaxSteps = 16` keeps the teaching code honest:
when the LLM (or our scripted stub) misbehaves, the loop terminates
with a clear error, not an infinite log file.

In production you'd combine a step cap with `context.WithTimeout`. We
do both in Phase G's integration tests.

## Why every `done_for_now` enters the thread

Upstream returns the done step without appending it to the thread (look
at the `if next_step.intent == "done_for_now"` branch). We append first,
then return. The reason: s12's replay test needs the full sequence of
decisions to reconstruct the agent's behaviour from a serialized thread.
If the terminal step is absent, replay can't tell whether the agent
finished or crashed.

## Reading map

- s05 → s06: HTTP handlers call `RunAgent` instead of main.
- s05 → s07: a non-done, non-tool intent (`request_more_information`)
  breaks the loop early. `RunAgent` learns one more exit branch.
- s05 → s12: `RunAgent` gets refactored into a `Reduce(Thread, Event)
  → Thread` shape.
