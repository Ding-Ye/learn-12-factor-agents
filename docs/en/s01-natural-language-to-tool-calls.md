---
title: "s01 · Minimum agent primitive — Natural language to tool calls"
chapter: 1
slug: s01-natural-language-to-tool-calls
est_read_min: 8
---

# s01 · Minimum agent primitive — Natural language to tool calls

> What you learn here: how an LLM returns something deterministic code can dispatch on — not the prompt, but the **return shape**. Everything in the remaining 11 chapters builds on the `NextStep` type defined here.

---

## Problem / The gap

Between natural language and code, we need an **intermediate format both sides agree on**. If we let the LLM emit free text "add 2 and 3", we end up writing regexes, parsers, and special cases for "3 plus 2" or "two plus three" — every prompt collapses into an NLP task.

The 12-factor answer: the LLM returns a **structured typed object** (a BAML class, a Python dataclass, a Go struct), and deterministic code uses a type-switch to pick the next move. This chapter doesn't yet call a real LLM — we first lock in the **return shape** and ship a stub provider so the whole pipeline is testable end-to-end. That foundation is what the other 11 chapters extend.

## Solution / Mental model

Three decisions worth pinning before any code:

1. **`Provider` is an interface**, not a direct Anthropic / OpenAI SDK call. From the stub here through Phase G's real providers, every chapter sees one signature: `DetermineNextStep(ctx, serialized) (NextStep, error)`.
2. **`NextStep` is a discriminated tagged union**: `Intent string` is the tag, `Data json.RawMessage` is the payload. The intent decides which payload struct to unmarshal into. This is the Go translation of BAML's `class XTool { intent "x"; ... }` pattern.
3. **`EchoProvider` is a deterministic stub**: it ignores its input and always returns the same `done_for_now` step. Tests in this chapter and the next ten don't need the network or an API key.

## How It Works

```
   argv ─► main ─► EchoProvider.DetermineNextStep(ctx, input)
                                        │
                                        ▼
                            NextStep{Intent, Data}
                                        │
                                        ▼
                  renderNextStep(step)  ──── switch step.Intent
                                              ├── "done_for_now" → unmarshal DoneForNowPayload
                                              └── default        → print raw JSON
                                        │
                                        ▼
                                     stdout
```

Core 30 lines (excerpt from [`agents/s01-natural-language-to-tool-calls/provider.go`](https://github.com/Ding-Ye/learn-12-factor-agents/blob/main/agents/s01-natural-language-to-tool-calls/provider.go)):

```go
type Provider interface {
    DetermineNextStep(ctx context.Context, serialized string) (NextStep, error)
}

type EchoProvider struct{}

const DefaultMessage = "Hello! How can I assist you today?"

func (EchoProvider) DetermineNextStep(_ context.Context, _ string) (NextStep, error) {
    data, err := json.Marshal(DoneForNowPayload{Message: DefaultMessage})
    if err != nil {
        return NextStep{}, err
    }
    return NextStep{
        Intent: "done_for_now",
        Data:   data,
    }, nil
}
```

**Three non-obvious bits:**

1. **`Data` is `json.RawMessage`, not `interface{}`** — `RawMessage` keeps the original bytes, so marshalling twice yields byte-identical output. With `interface{}`, a re-marshal of a `map[string]interface{}` would have an unstable key order. When s06 starts shipping `NextStep` over HTTP, that stability matters.
2. **`DetermineNextStep` takes a `serialized string`** — not a `Thread`, not a `[]Message`. In s01 the parameter is ignored, but pinning it as `string` means s03's introduction of `Thread` doesn't change the interface — only the implementation. Provider always sees pre-serialized text.
3. **The `error` return looks redundant** — `EchoProvider` cannot fail. But the interface needs to leave room for the real providers introduced later (and in Phase G): timeouts, HTTP errors, JSON-decoding failures all need a way out.

## What Changed vs prior chapter

s01 is the bootstrap chapter; there is no s00. What we ship is the baseline:

- `types.go` — `NextStep` + `DoneForNowPayload`
- `provider.go` — `Provider` interface + `EchoProvider`
- `main.go` — CLI entry + `renderNextStep` dispatch
- `provider_test.go` — 5 tests covering the wire format

The next eleven chapters evolve these four files in shape, not in spirit.

## Try It

```bash
cd agents/s01-natural-language-to-tool-calls

# Run tests (no network needed)
go test -v ./...

# Default greeting
go run . "hello"

# Any input returns the same done_for_now
go run . "add 5 and 3, then multiply by 2"
```

Expected output shape:

```
intent=done_for_now message="Hello! How can I assist you today?"
```

Test output: 5 PASS, covering (1) intent is always `done_for_now`, (2) message is the canonical string, (3) `NextStep` round-trips through JSON, (4) `renderNextStep` happy path, (5) `renderNextStep` fallback for unknown intents.

## Upstream Source Reading

The upstream equivalent splits across **BAML** (`workshops/2025-07-16/walkthrough/01-agent.baml`) and **Python** (`workshops/2025-07-16/walkthrough/01-agent.py`). BAML is upstream's DSL for codegen'd typed LLM output — it does the same job as our `NextStep{Intent, Data}`, just via a build step instead of by-hand.

```upstream:workshops/2025-07-16/walkthrough/01-agent.baml#L1-L27
// Source: workshops/2025-07-16/walkthrough/01-agent.baml lines 1-27
// BAML syntax: declare a function returning a typed union

class DoneForNow {
  intent "done_for_now"      // ← literal type, equivalent to Go's `Intent string`
  message string              // ← our DoneForNowPayload.Message
}

// This BAML function declaration ≈ our Provider.DetermineNextStep signature
function DetermineNextStep(
    thread: string             // ← upstream also takes a string,
) -> DoneForNow {              //    confirming our `serialized string` choice
    client Qwen3               // ← BAML-side LLM client config
    prompt #"
        {{ _.role("system") }}
        You are a helpful assistant that responds to the user's message.
        {{ _.role("user") }}
        You are given the following thread of events:
        {{ thread }}
    "#
}
```

```upstream:workshops/2025-07-16/walkthrough/01-agent.py#L23-L26
# Source: workshops/2025-07-16/walkthrough/01-agent.py lines 23-26
# Python call site — analogous to our main.go's EchoProvider.DetermineNextStep call
def agent_loop(thread: Thread) -> AgentResponse:
    b = get_baml_client()
    next_step = b.DetermineNextStep(thread.serialize_for_llm())
    return next_step
```

**Reading notes:**

- **BAML codegen vs hand-written struct**: upstream compiles BAML to a typed Python class; we hand-write `DoneForNowPayload` + `json.RawMessage`. Functionally equivalent, but BAML adds a build step. Our pick saves Go learners one toolchain.
- **`prompt` block**: upstream embeds the prompt directly inside the BAML file; we don't introduce prompt templating until s02 (which uses Go's `text/template`).
- **`client Qwen3`**: upstream configures its LLM client inside BAML; we use a stub in s01 and don't introduce a real client until Phase G.
- **`thread: string` parameter**: upstream and we both pass a string — confirming "Provider always sees serialized text" is upstream's real design, not a Go simplification.
- **What we omit**: `Event` / `Thread` / `serialize_for_llm` from `01-agent.py` belong to s03. s01 stays minimal: just `NextStep`.

**Want to read more?** Start at `workshops/2025-07-16/walkthrough/01-agent.baml`, follow `DetermineNextStep` into `01-agent.py`, then read `01-main.py` for the call site. That trace mirrors our s01 → s03 → s04 progression.

---

**Up next, s02:** the "prompt-is-a-framework-black-box" critique gets unpacked. We render the prompt explicitly via Go's `text/template`, and the provider receives a rendered string instead of raw input. `EchoProvider` is upgraded to record the prompt hash so tests can assert the template actually ran.
