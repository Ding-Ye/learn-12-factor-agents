---
title: "s02 · Own your prompts"
chapter: 2
slug: s02-own-your-prompts
est_read_min: 7
---

# s02 · Own your prompts

> What you learn here: stop letting the framework hide the prompt. Every token sent to the LLM comes from a template you wrote, and tests can prove the template actually ran.

---

## Problem / The gap

s01 nailed the wire format (`NextStep` + `Provider`), but the prompt is still a mystery — `EchoProvider` ignores its input and returns a fixed reply. In real code, model reliability is almost entirely a function of the prompt: one wrong token and the output drifts; tweak a role marker and the model loses the thread.

Upstream factor-02's argument is blunt: **the prompt is code**. Don't let langchain / crewai assemble the system message for you — when it breaks you can neither see the final prompt nor edit any of its parts. s02 lands the principle in Go: render with `text/template` explicitly, and verify via a `prompt_hash` that the rendering actually reached the provider.

## Solution / Mental model

Three decisions:

1. **`promptTemplate` is a Go string constant**, structured to mirror upstream's BAML system/user split. Reading that single declaration tells you everything the LLM sees.
2. **`RenderPrompt(PromptInput) (string, error)`** is the only render entrypoint. `PromptInput` is a struct (not a map), so missing fields are compile errors.
3. **`RecordingProvider`** replaces s01's `EchoProvider`. It stashes the received prompt in `LastSeen` and embeds `prompt_hash=...` in the `done_for_now` message. Tests compare the hash against expectations — change the template, the hash changes.

## How It Works

```
   argv ─► RenderPrompt(PromptInput) ─► rendered string (SYSTEM:/USER:)
                                                   │
                                                   ▼
                          RecordingProvider.DetermineNextStep(ctx, rendered)
                                          │
                                          ├─ stores rendered into .LastSeen
                                          ▼
                          NextStep{Intent:"done_for_now",
                                   Data:{message:"Acknowledged. prompt_hash=<sha8>"}}
                                          │
                                          ▼
                                  renderNextStep ─► stdout
```

Core 30 lines (excerpt from [`agents/s02-own-your-prompts/prompt.go`](https://github.com/Ding-Ye/learn-12-factor-agents/blob/main/agents/s02-own-your-prompts/prompt.go)):

```go
const promptTemplate = `SYSTEM:
You are a helpful assistant that responds to the user's message.

USER:
You are given the following thread of events:
{{ .UserInput }}

What should the next step be?
`

type PromptInput struct {
    UserInput string
}

func RenderPrompt(in PromptInput) (string, error) {
    t, err := template.New("agent-prompt").Parse(promptTemplate)
    if err != nil { return "", fmt.Errorf("parse template: %w", err) }
    var buf bytes.Buffer
    if err := t.Execute(&buf, in); err != nil {
        return "", fmt.Errorf("execute template: %w", err)
    }
    return buf.String(), nil
}
```

**Three non-obvious bits:**

1. **`PromptInput` is a struct, not a map** — `map[string]any` would silently render the empty string when a template references a missing key. Struct fields turn typos like `{{ .UserInpit }}` into explicit `Execute` errors.
2. **`PromptHash` keeps only the first 8 bytes** — a full sha256 is 32 bytes and unreadable. Eight bytes = 16 hex chars, plenty for human comparison. The trade-off is collision probability (~10^-19), which is fine for a teaching demo.
3. **`RecordingProvider.LastSeen` is an exported field** — on purpose. Tests need to inspect the full rendered prompt when the hash check fails (to locate the diff). Production code would make it unexported with a getter; here readability beats encapsulation.

## What Changed vs s01

```diff
+ prompt.go               (new — RenderPrompt + PromptHash)
  types.go                (NextStep + DoneForNowPayload unchanged)
- provider.go: EchoProvider
+ provider.go: RecordingProvider  (.LastSeen + embed hash)
  main.go                 (one extra step: RenderPrompt before provider call)
- 3 tests (s01)
+ 6 tests (s02)
```

Semantically: s01's provider was an "ignore-input" stub; s02's is a "must see the rendered prompt" stub. Every subsequent chapter's provider keeps that contract.

## Try It

```bash
cd agents/s02-own-your-prompts

go test -v ./...

go run . "add 5 and 3"
# → intent=done_for_now message="Acknowledged. prompt_hash=<16-hex-chars>"

go run . "different input"
# Different hash — proves RenderPrompt actually rendered different content
```

Expected output shape:

```
intent=done_for_now message="Acknowledged. prompt_hash=03b6f1c2afe0945b"
```

Different inputs render different prompts, so their hashes differ. Edit `promptTemplate` and re-run tests: `TestPromptHash_Stable` locks "same input → same hash," and `TestRenderPrompt_HasRoleMarkers` checks the structure.

## Upstream Source Reading

Upstream puts the prompt inside the BAML file:

```upstream:workshops/2025-07-16/walkthrough/01-agent.baml#L11-L27
// Source: workshops/2025-07-16/walkthrough/01-agent.baml lines 11-27
function DetermineNextStep(
    thread: string
) -> DoneForNow {
    client Qwen3
    prompt #"
        {{ _.role("system") }}
        You are a helpful assistant that responds to the user's message.

        {{ _.role("user") }}
        You are given the following thread of events:
        {{ thread }}

        What should the next step be?
    "#
}
```

**Reading notes:**

- **BAML `{{ _.role("system") }}` vs Go `SYSTEM:`** — BAML compiles the marker into the Anthropic API's `messages[].role` field. Until we wire a real provider (Phase G), plain text `SYSTEM:` / `USER:` is enough. Phase G's real provider parses these markers back into a proper messages array.
- **BAML `{{ thread }}` vs Go `{{ .UserInput }}`** — upstream already introduced `Thread` (a string returned from `serialize_for_llm`); we don't yet, so we use raw `UserInput`. s03 swaps this for `{{ .Thread }}`.
- **`prompt #"..."#`** — BAML's multi-line string syntax, equivalent to Go's `` ` `` raw string.
- **`client Qwen3`** — BAML puts LLM client config next to the prompt. We push that responsibility into the `Provider` implementation (here `RecordingProvider`; Phase G plugs in real clients).
- **What we omit** — upstream BAML can also auto-inject typed-output JSON Schema with `{{ ctx.output_format }}`. We hand-write that in s04 once we have multiple tool intents.

**Want to read more?** `content/factor-02-own-your-prompts.md:14-91` contains the full "why prompts are code" argument. Worth reading in full — it's one of the manifesto's purest pieces.

---

**Up next, s03:** the "input" graduates from a single string to a `Thread{Events []Event}`. The provider receives `json.Marshal(thread.Events)` instead of `UserInput`. That's how 12-factor's "own your context window" lands in code.
