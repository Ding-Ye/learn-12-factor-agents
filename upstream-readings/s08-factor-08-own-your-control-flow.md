# Upstream reading — Factor 8: Own your control flow

> Source: humanlayer/12-factor-agents @ d20c728.

## Source: `workshops/2025-07-16/walkthrough/07-agent.py` (38-80)

```python
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
```

## Mapping

| Upstream pattern | Our Go translation |
|---|---|
| `if/elif` per intent | `Action` enum + `ControlFlow` switch |
| Inline `await ...` on async branches | `ActionLoop` calls `tool.Execute` synchronously |
| Implicit `else` (no branch → re-loop) | `ActionEscalate` + error event |
| Adding a new intent = new elif | Adding a new intent = new `Action` case (or fall through to Escalate) |

## Why the Go port is more verbose

Python's duck typing + flexible control flow make `if/elif` chains
natural. Go's type system rewards explicit enums + pure functions. The
extra ceremony is the cost of buying static guarantees (exhaustiveness,
testability, "did I forget a branch?").

## Reading map

- s08 → s09: `error` event becomes a first-class branch — when
  `tool.Execute` returns an error, append the error event and decide
  next action based on consecutive error count.
- s08 → s10: orchestrator + sub-agents introduce a new kind of
  "tool" — `Action` doesn't change, but the loop now also dispatches
  "sub-agent invocation" intents.
- s08 → s12: ControlFlow becomes the core of a pure `Reduce(Thread,
  Event) → Thread` function.
