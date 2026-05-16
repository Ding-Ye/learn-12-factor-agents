---
title: "s04 · Tools are structured outputs"
chapter: 4
slug: s04-tools-are-structured-outputs
est_read_min: 8
---

# s04 · Tools are structured outputs

> What you learn here: LLM output graduates from free text to a typed tool struct. `NextStep.Data` carries its first typed payload, and `main.go` gets its first `switch step.Intent` dispatcher.

---

## Problem / The gap

s03 got the context right, but the LLM still only returns `done_for_now`. A real agent needs the model to emit "add 2 + 3" or "call this API" as a **typed action** that code can dispatch immediately. If we let the model emit free text, every prompt collapses into NLP — write a parser, handle "3 plus 2" and "two plus three," etc.

Upstream factor-04's answer: **each tool is a typed struct**. The LLM sees a schema, emits JSON matching it, and a type-switch dispatches. This chapter introduces the `Tool` interface + five concrete tools (add/subtract/multiply/divide/done_for_now) + a `Registry` for lookup.

## Solution / Mental model

Three decisions:

1. **`Tool` has two methods**: `Intent() string` returns the discriminator, `Execute(ctx, json.RawMessage) (any, error)` runs the work. Minimal surface keeps "add a new tool" cheap (one struct, two methods).
2. **`MathPayload` is shared by the four math tools**: upstream BAML defines four separate classes (AddTool / SubtractTool / MultiplyTool / DivideTool) with identical fields. We share `MathPayload{A,B}` — the discriminator lives on `NextStep.Intent`, not the payload.
3. **`Registry` is `map[string]Tool`**: O(1) dispatch; adding a tool means editing one place (`DefaultRegistry()`).

## How It Works

```
   argv ──► ScriptedProvider.DetermineNextStep ──► NextStep{Intent:"add", Data:{a,b}}
                                                              │
                                                              ▼
                                          Registry.Lookup(step.Intent) ──► AddTool{}
                                                              │
                                                              ▼
                                          AddTool.Execute(ctx, step.Data) ──► 5.0
                                                              │
                                                              ▼
                                                    print result
```

Core 30 lines (excerpt from [`agents/s04-tools-are-structured-outputs/tools.go`](https://github.com/Ding-Ye/learn-12-factor-agents/blob/main/agents/s04-tools-are-structured-outputs/tools.go)):

```go
type Tool interface {
    Intent() string
    Execute(ctx context.Context, payload json.RawMessage) (any, error)
}

type AddTool struct{}

func (AddTool) Intent() string { return IntentAdd }
func (AddTool) Execute(_ context.Context, payload json.RawMessage) (any, error) {
    var p MathPayload
    if err := json.Unmarshal(payload, &p); err != nil {
        return nil, fmt.Errorf("decode add payload: %w", err)
    }
    return p.A + p.B, nil
}

type Registry map[string]Tool

func DefaultRegistry() Registry { /* ... */ }
func (r Registry) Lookup(intent string) (Tool, error) { /* ... */ }
```

**Three non-obvious bits:**

1. **`Tool.Execute` doesn't take Thread** — tools are pure `(input → output)`. Thread accumulation is the loop's job (introduced in s05). This separation lets us unit-test tools cleanly and share the same tool pool across sub-agents (s10).
2. **`done_for_now` doesn't implement `Tool`** — it's a loop-exit signal, not an action. `main.go` special-cases it before dispatch. s05 makes this special case explicit via `ControlFlow`.
3. **`DivideTool`'s error string is verbatim from upstream** — `"Error: Division by zero"` mirrors `05-agent.py:51-52`. s09 turns errors into thread events, and that exact string becomes the LLM's evidence for self-correction.

## What Changed vs s03

```diff
+ types.go: MathPayload, intent constants
+ tools.go (new — Tool interface + 4 math tools + Registry)
- provider.go: EchoThreadProvider
+ provider.go: ScriptedProvider (emits non-done intent based on input keywords)
- main.go: single render
+ main.go: dispatch (Registry.Lookup + Tool.Execute)
- 6 tests
+ 7 tests
```

Semantically: s03's provider always returned `done_for_now`; s04's emits a typed non-done `NextStep` based on input. Dispatch is still a single step — no loop yet.

## Try It

```bash
cd agents/s04-tools-are-structured-outputs

go test -v ./...

go run . "add 2 and 3"
# → intent=add payload={"a":2,"b":3} result=5

go run . "multiply 4 and 6"
# → intent=multiply payload={"a":4,"b":6} result=24

go run . "say hi"
# → intent=done_for_now message="Nothing to do."
```

## Upstream Source Reading

```upstream:workshops/2025-07-16/walkthrough/05-agent.baml#L1-L37
// Source: workshops/2025-07-16/walkthrough/05-agent.baml lines 1-37
// License: Apache 2.0

class AddTool {
  intent "add"
  a int | float
  b int | float
}

class SubtractTool {
  intent "subtract"
  a int | float
  b int | float
}

class MultiplyTool {
  intent "multiply"
  a int | float
  b int | float
}

class DivideTool {
  intent "divide"
  a int | float
  b int | float
}

class DoneForNow {
  intent "done_for_now"
  message string
}

function DetermineNextStep(thread: string)
    -> DoneForNow | AddTool | SubtractTool | MultiplyTool | DivideTool {
    client Qwen3
    prompt #" ... "#
}
```

**Reading notes:**

- **BAML union return vs our `NextStep`**: BAML encodes "returns one of five tools" in the function signature (`-> A | B | C | D | E`), giving the LLM a schema constraint. Our Go port uses `NextStep{Intent, Data}` + a `Registry` for runtime validation. Functionally equivalent; BAML's compile-time guarantee is stronger.
- **`a int | float` vs `A float64`**: BAML distinguishes int/float (union types); our Go port unifies on `float64` for simplicity. Upstream Python's `MathPayload` does the same one-type collapse.
- **Each class repeats `a`, `b`**: BAML classes don't share fields via inheritance; we share `MathPayload` across the four math tools.
- **divide-by-zero handling**: upstream `05-agent.py:51-52` catches at execute time and returns the string "Error: Division by zero". We return a Go `error`; `main.go` decides how to render. s09 starts putting these errors into the thread.
- **`client Qwen3`**: BAML targets Baseten-hosted Qwen3 32B. Phase G demonstrates swapping to OpenAI/Anthropic.

**Want to read more?** `content/factor-04-tools-are-structured-outputs.md:11-50` argues why "tool = returns typed struct" is the key abstraction. Worth reading in full.

---

**Up next, s05:** `main.go`'s single-step dispatch becomes a real agent loop — `RunAgent(thread, provider, tools)` calls provider → execute → append repeatedly until `done_for_now`. The Thread carries multiple tool_call / tool_response events for the first time.
