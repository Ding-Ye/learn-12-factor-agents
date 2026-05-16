# Upstream reading — Factor 4: Tools are structured outputs

> Annotated walkthrough of the upstream BAML tool definitions s04 mirrors.
> Source: humanlayer/12-factor-agents @ d20c728.

## Source: `workshops/2025-07-16/walkthrough/05-agent.baml` (lines 1-37)

```baml
// Source: workshops/2025-07-16/walkthrough/05-agent.baml lines 1-37
// License: Apache 2.0

class AddTool {        // ← each tool is its own class
  intent "add"          // ← intent literal is the discriminator
  a int | float         // ← union type — BAML supports both
  b int | float
}

class SubtractTool { intent "subtract"; a int|float; b int|float }
class MultiplyTool { intent "multiply"; a int|float; b int|float }
class DivideTool   { intent "divide";   a int|float; b int|float }

class DoneForNow {
  intent "done_for_now"
  message string
}

// The return type is a union of all possible tool classes.
function DetermineNextStep(thread: string)
    -> DoneForNow | AddTool | SubtractTool | MultiplyTool | DivideTool {
    client Qwen3
    prompt #"..."#
}
```

## Mapping table

| Upstream concept | Our Go equivalent | File |
|---|---|---|
| `class XTool { intent "x"; ... }` | one `struct` per tool + `Intent()` method | `tools.go` |
| BAML union return type | `NextStep{Intent string; Data json.RawMessage}` | `types.go` |
| BAML's schema injection to LLM | Phase G's real-provider prompt (Go side: hand-written JSON schema string) | (TBD Phase G) |
| `client Qwen3` | `Provider` implementation choice | `provider.go` |
| Per-tool execution in Python | `Tool.Execute(ctx, payload) (any, error)` | `tools.go` |

## Why a Registry, not a switch

A bare `switch step.Intent { case IntentAdd: ... }` works but:

- adding a tool means editing both the switch and somewhere else
- can't reuse the dispatch logic across sub-agents (s10)
- harder to enumerate "what tools are available" at runtime (Phase G
  needs this to build the JSON schema injected into the prompt)

A `Registry map[string]Tool` solves all three. The cost is one extra
indirection at runtime; we trade that for code organization.

## Why `MathPayload` is shared

BAML defines four separate classes with identical fields. The reason is
BAML doesn't have struct embedding/inheritance — each class declaration
stands alone, which is fine because BAML is a DSL not a general-purpose
language.

Go has struct embedding, but we don't need it: the discriminator lives on
`NextStep.Intent`, not on the payload. One `MathPayload{A,B}` per math
tool is enough. (We do lose the ability to put per-tool documentation on
the payload struct, but tool-level documentation lives on the `Tool`
implementation struct itself, so this is a wash.)

## Reading map

- s04 → s05: the loop wraps multiple `Registry.Lookup` + `Tool.Execute`
  calls into one `RunAgent` function.
- s04 → s07: the `Tool` interface gets two more implementations
  (`RequestApproval`, `AskClarification`) that don't actually execute
  but instead break the loop.
- s04 → s09: tool `error` returns get translated to `error` events on
  the thread; the LLM reads them and self-corrects.
- Phase G: the JSON schemas we'd auto-generate from BAML (`{{ ctx.output_format }}`)
  become hand-written prompt fragments injected into the system prompt.
