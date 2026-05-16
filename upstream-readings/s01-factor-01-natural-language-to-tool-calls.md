# Upstream reading — Factor 1: Natural language to tool calls

> Annotated walkthrough of the three upstream files s01 of our learn repo
> draws from. Quotations are from
> https://github.com/humanlayer/12-factor-agents @ commit
> `d20c728368bf9c189d6d7aab704744decb6ec0cc`.
>
> Code under Apache 2.0; conceptual prose under CC-BY-SA 4.0
> (https://creativecommons.org/licenses/by-sa/4.0/).

## 1. `content/factor-01-natural-language-to-tool-calls.md`

The conceptual side. Key passages we'll reuse in `docs/{zh,en}/s01-*.md`:

> Probably the most common pattern in agent building is to convert
> natural language to structured tool calls. This is a powerful pattern
> that allows you to build agents that can reason about a task and take
> action.
>
> _(from `content/factor-01-natural-language-to-tool-calls.md:5-9`)_

What the factor demands of an implementation:

- a closed set of intents (called "tools" in upstream parlance)
- one of those intents is the "I'm done" signal (the `DoneForNow` case)
- the dispatcher pattern-matches on the intent literal

What we mirror in Go:

| Upstream concept | Our type | File |
|---|---|---|
| Tool union return | `NextStep{Intent, Data}` | `agents/s01-natural-language-to-tool-calls/types.go` |
| `DoneForNow` class | `DoneForNowPayload` | `agents/s01-natural-language-to-tool-calls/types.go` |
| BAML-generated client | `Provider` interface | `agents/s01-natural-language-to-tool-calls/provider.go` |
| `b.DetermineNextStep(...)` | `EchoProvider.DetermineNextStep` | `agents/s01-natural-language-to-tool-calls/provider.go` |

What we **don't** mirror in s01 (deferred to later chapters):

- prompt templating (s02)
- `Event` / `Thread` / `serialize_for_llm` (s03)
- multiple tool intents (s04)
- the agent loop (s05)

## 2. `workshops/2025-07-16/walkthrough/01-agent.baml`

The full BAML file is 27 lines. Annotated:

```baml
// Lines 1-9 — the discriminated union member for "done":
class DoneForNow {
  intent "done_for_now"   // string literal type — the discriminator
  message string          // payload field
}

// Lines 11-27 — the function whose return type IS the union.
// Our Provider.DetermineNextStep signature is the Go translation.
function DetermineNextStep(
    thread: string         // pre-serialized history
) -> DoneForNow {          // return type would widen in later .baml files
    client Qwen3           // LLM client configuration lives in BAML
    prompt #"
        {{ _.role("system") }}
        You are a helpful assistant that responds to the user's message.
        {{ _.role("user") }}
        You are given the following thread of events:
        {{ thread }}
    "#
}
```

Differences from our Go port:

1. BAML is a separate DSL with its own compiler (`baml-cli generate`).
   Our `types.go` is plain Go, no codegen.
2. The `prompt` block is embedded in the function declaration. We move
   that to a `text/template` in s02.
3. `client Qwen3` is a BAML-side concern. Our provider implementations
   own client configuration (stub in s01; Phase G adds real clients).
4. BAML's `string` literal types (e.g., `intent "done_for_now"`) give
   compile-time discrimination. In Go we settle for runtime checks on
   `step.Intent` plus tests that document the closed set.

## 3. `workshops/2025-07-16/walkthrough/01-agent.py`

The full file is 26 lines. Annotated:

```python
import json
from typing import Dict, Any, List

# `AgentResponse` is loosely typed in the upstream Python; we tighten it
# into the NextStep struct in our Go port.
AgentResponse = Any

# Event and Thread are defined here but unused in s01 — s03 of our
# curriculum is where they earn their keep.
class Event:
    def __init__(self, type: str, data: Any):
        self.type = type
        self.data = data

class Thread:
    def __init__(self, events: List[Dict[str, Any]]):
        self.events = events

    def serialize_for_llm(self):
        # Default JSON; later chapters demo XML for token cost.
        return json.dumps(self.events)

# The call site we mirror in main.go.
def agent_loop(thread: Thread) -> AgentResponse:
    b = get_baml_client()
    next_step = b.DetermineNextStep(thread.serialize_for_llm())
    return next_step
```

Differences from our Go port (for s01 only):

1. Upstream's `Event` / `Thread` exist in this file but go unused until
   later chapters. We defer their definition to s03 to keep s01 truly
   minimal.
2. `get_baml_client()` is a runtime indirection; our Go code instantiates
   `EchoProvider{}` directly.
3. The Python returns `AgentResponse = Any`. Our Go version returns
   `(NextStep, error)`; the `error` slot is unused in s01 but reserved
   for s02+.
4. No `if __name__ == "__main__"` block here — that's in `01-main.py`,
   which we mirror in our `main.go`.

## Why the divergence matters

Upstream's pedagogy assumes the reader is comfortable jumping between
BAML, Python, and TypeScript across the same chapter. Our Go port
collapses that into one language so the **shape** of each factor stands
out without the cross-language tax.

The shape we lock in s01:

```
INPUT  (string) ──► Provider.DetermineNextStep ──► NextStep{Intent, Data} ──► dispatcher
```

This shape doesn't change for the remaining eleven chapters. Only the
implementation behind `Provider`, the contents of `Thread`, and the size
of the dispatcher do.
