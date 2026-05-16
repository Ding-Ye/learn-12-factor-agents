---
title: "s03 · Own your context window (Thread + Event)"
chapter: 3
slug: s03-own-your-context-window
est_read_min: 8
---

# s03 · Own your context window (Thread + Event)

> What you learn here: the "input" graduates from a single string to a stream of events. `Thread{Events []Event}` is the central data structure that powers every chapter to follow.

---

## Problem / The gap

s02 made the prompt explicit, but the only content embedded was a raw user message. A real agent run produces user_input → tool_call → tool_response → tool_call → … — every step needs to enter the prompt. If the prompt only has "the original question," the model can neither see what just happened nor self-correct after errors.

Upstream factor-03's answer: **store the entire interaction history as a list of events**, serialize it, and embed in the prompt. This chapter pins `Event{Type, Data}` + `Thread{Events []Event}` + `SerializeForLLM()`. The next nine chapters extend the structure (tool calls, error events, human responses, etc.).

## Solution / Mental model

Three decisions:

1. **`Event.Type` is a const string**, not an int enum — after JSON serialization the LLM sees `"type":"user_input"` instead of `"type":1`, which is far more readable.
2. **Constructors over literals**: write `NewUserInputEvent("hi")` instead of `Event{Type:"user_input", Data:"hi"}`. Centralizing constructors in `events.go` makes the closed set visible.
3. **`SerializeForLLM()` defaults to indented JSON**: readability first. s07 demonstrates an XML variant for token cost.

## How It Works

```
   argv ──► NewUserInputEvent("...") ──► Thread{Events:[...]}
                                                  │
                                                  ▼
                                  Thread.SerializeForLLM()  → JSON string
                                                  │
                                                  ▼
                            RenderPrompt(PromptInput{Thread: ...})
                                                  │
                                                  ▼
                            EchoThreadProvider.DetermineNextStep(ctx, rendered)
                                                  │
                                                  ▼
                       NextStep{Intent:"done_for_now",
                                Data:{message:"Thread received with N user_input events."}}
```

Core 30 lines (excerpt from [`agents/s03-own-your-context-window/thread.go`](https://github.com/Ding-Ye/learn-12-factor-agents/blob/main/agents/s03-own-your-context-window/thread.go)):

```go
type Event struct {
    Type string      `json:"type"`
    Data interface{} `json:"data"`
}

type Thread struct {
    Events []Event
}

func NewThread(seed ...Event) *Thread {
    t := &Thread{Events: make([]Event, 0, 8)}
    t.Events = append(t.Events, seed...)
    return t
}

func (t *Thread) Append(e Event)            { t.Events = append(t.Events, e) }
func (t *Thread) LastEvent() (Event, bool)  { /* ... */ }
func (t *Thread) SerializeForLLM() string   { /* json.MarshalIndent */ }
```

**Three non-obvious bits:**

1. **`make([]Event, 0, 8)` pre-allocation** — typical agent runs hit < 8 steps; preallocating 8 capacity avoids the first few `append` reallocations. Micro-optimization but readable and matches the 12-factor "small focused" ethos.
2. **`Data interface{}` round-trip widening** — on write, `Data` may be `string` / `map[string]any` / a typed struct. After JSON unmarshal it's always `map[string]any` (or primitive). This is Go's runtime type erasure; s12's reducer tests cover the implications.
3. **`SerializeForLLM` returns `string`, not `([]byte, error)`** — the Provider interface takes a `string`, so we keep call sites symmetric. The fallback path returns a placeholder string instead of panicking so test diffs stay useful.

## What Changed vs s02

```diff
+ events.go      (new — Event + 3 constructors)
+ thread.go      (new — Thread + Append/LastEvent/SerializeForLLM)
  types.go       (NextStep unchanged)
- prompt.go: PromptInput.UserInput
+ prompt.go: PromptInput.Thread
- provider.go: RecordingProvider
+ provider.go: EchoThreadProvider (counts user_input events)
  main.go        (seed thread → render → provider)
```

Semantically: s02's provider saw "one sentence + template wrap"; s03 sees "serialized event stream + same template wrap." The interface signature (`string → NextStep`) did not change.

## Try It

```bash
cd agents/s03-own-your-context-window

go test -v ./...

go run . "hello"
# → intent=done_for_now message="Thread received with 1 user_input event."

# Multiple user_input events can't be triggered from the CLI (it seeds
# only one) — tests cover the multi-event case.
go test -run TestEchoThreadProvider -v
```

Expected output shape: `intent=done_for_now message="Thread received with 1 user_input event."`

Test coverage: 6 PASS (event order on append, byte-stable serialization, `LastEvent`, prompt rendering, event-count detection, JSON round-trip).

## Upstream Source Reading

```upstream:workshops/2025-07-16/walkthrough/01-agent.py#L1-L26
# Source: workshops/2025-07-16/walkthrough/01-agent.py lines 1-26
# License: Apache 2.0

import json
from typing import Dict, Any, List

AgentResponse = Any

class Event:
    def __init__(self, type: str, data: Any):
        self.type = type
        self.data = data

class Thread:
    def __init__(self, events: List[Dict[str, Any]]):
        self.events = events

    def serialize_for_llm(self):
        # can change this to whatever custom serialization you want to do, XML, etc
        return json.dumps(self.events)

def agent_loop(thread: Thread) -> AgentResponse:
    b = get_baml_client()
    next_step = b.DetermineNextStep(thread.serialize_for_llm())
    return next_step
```

**Reading notes:**

- **Constructors**: Python writes `Event(type, data)`; we write `NewUserInputEvent("...")`. Python leaves the type string to the caller; we close the set with constants.
- **`thread.events` is list of dict, not list of Event**: upstream's Thread takes a list of dicts — events are pre-serialized to `Dict[str, Any]`. Our Go Thread carries typed `Event` values and lets JSON do the dict transform at serialize time.
- **`serialize_for_llm()`**: upstream defaults to JSON with a hint that XML is also fine — we demonstrate XML in s07.
- **`AgentResponse = Any`**: upstream gives up on typing; our Go port keeps `NextStep` typed. Free win Go gets over Python.
- **What we omit**: upstream's `01-agent.py` already includes `agent_loop` — but it's single-shot. We introduce the real loop in s05 to keep this chapter focused on the data structure.

**Want to read more?** `content/factor-03-own-your-context-window.md` is worth reading in full, especially lines 69-112 on the XML serialization trade-off. s07 picks up that thread.

---

**Up next, s04:** the Provider stops always returning `done_for_now`. We introduce `AddTool` / `SubtractTool` / … as structured outputs. NextStep's `Data` starts carrying typed payloads like `{"a":2,"b":3}`, and `main.go` gets its first `switch step.Intent` dispatcher.
