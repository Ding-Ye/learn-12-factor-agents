# Upstream reading — Factor 3: Own your context window

> Annotated walkthrough of the upstream Event + Thread types that s03
> mirrors. Source: https://github.com/humanlayer/12-factor-agents @
> commit `d20c728368bf9c189d6d7aab704744decb6ec0cc`.

## Source: `workshops/2025-07-16/walkthrough/01-agent.py` (lines 1-26)

```python
# Source: workshops/2025-07-16/walkthrough/01-agent.py lines 1-26
# License: Apache 2.0

import json
from typing import Dict, Any, List

AgentResponse = Any  # ← upstream's `NextStep` is loosely typed

class Event:
    # The closed set of types is by convention, not enforcement.
    # Our Go port adds const Event Type strings in events.go.
    def __init__(self, type: str, data: Any):
        self.type = type
        self.data = data

class Thread:
    def __init__(self, events: List[Dict[str, Any]]):
        # Note: events arrives as `List[Dict[str, Any]]`, not List[Event].
        # Upstream's data hygiene here is loose — but it works because
        # the only consumer is `serialize_for_llm` which json.dumps it.
        self.events = events

    def serialize_for_llm(self):
        # JSON is the default; comment mentions XML as an alternative.
        # We demonstrate XML in s07.
        return json.dumps(self.events)

def agent_loop(thread: Thread) -> AgentResponse:
    b = get_baml_client()
    next_step = b.DetermineNextStep(thread.serialize_for_llm())
    return next_step
```

## Mapping table

| Upstream concept | Our Go equivalent | File |
|---|---|---|
| `class Event(type, data)` | `Event{Type, Data}` | `agents/s03-*/events.go` |
| `class Thread([events])` | `Thread{Events []Event}` | `agents/s03-*/thread.go` |
| `serialize_for_llm()` | `(*Thread).SerializeForLLM()` | `agents/s03-*/thread.go` |
| `agent_loop` (single-shot) | `main.go` (no loop yet — s05) | `agents/s03-*/main.go` |
| `AgentResponse = Any` | `NextStep struct` | `agents/s03-*/types.go` |

## Why our Thread takes `[]Event` instead of `[]map[string]any`

Two reasons:

1. Type safety at the write site. `thread.Append(NewUserInputEvent("hi"))`
   catches typos. `thread.Append(map[string]any{"typ": "user_input"})`
   doesn't (note the typo).
2. Constructors centralize the closed Event Type set. Adding a new event
   type means adding a new constructor in `events.go` — grep-able.

The cost is that `Event.Data` is `interface{}`, which widens on JSON
round-trip. We document this and revisit in s12's reducer.

## Reading map

- s04 adds tool-call payloads to `Event.Data` (still via constructors).
- s05 introduces the loop and starts appending events programmatically.
- s07 demonstrates a non-JSON serializer (XML).
- s12 makes the whole Reduce-over-Events explicit.

## Quoted prose

From `content/factor-03-own-your-context-window.md` lines 121-139
(CC-BY-SA 4.0; attributed):

> Most agents have a context window that grows as they reason. ... If
> you don't control the format, the model has no way to learn from past
> errors — they get summarized away by some hidden framework layer.
> Own it.
