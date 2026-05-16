# Upstream reading — Factor 2: Own your prompts

> Annotated walkthrough of the upstream prompt block s02 mirrors. Source:
> https://github.com/humanlayer/12-factor-agents @ commit
> `d20c728368bf9c189d6d7aab704744decb6ec0cc`.
>
> Code under Apache 2.0; quoted prose under CC-BY-SA 4.0.

## 1. `content/factor-02-own-your-prompts.md`

The thesis (paraphrased from lines 14-30):

> Frameworks that "manage prompts for you" leave you blind when the model
> misbehaves. You can't iterate on what you can't see. Own every token
> reaching the LLM.

What this demands of an implementation:

- the prompt is a string template **in your repo** (not in a vendored
  library, not behind an SDK call)
- changes to the template are diffable via git
- tests can assert structural properties of the rendered prompt
  (sections present, ordering correct)

We satisfy these in Go with:

| Upstream concept | Our type | File |
|---|---|---|
| Inline prompt block in BAML | `promptTemplate` const | `agents/s02-own-your-prompts/prompt.go` |
| `{{ _.role("system") }}` marker | `SYSTEM:` line | `agents/s02-own-your-prompts/prompt.go` |
| `{{ thread }}` substitution | `{{ .UserInput }}` (s03 widens to `{{ .Thread }}`) | `agents/s02-own-your-prompts/prompt.go` |
| BAML-generated client wrapper | `RecordingProvider` | `agents/s02-own-your-prompts/provider.go` |

## 2. `workshops/2025-07-16/walkthrough/01-agent.baml` lines 11-27

```baml
// Source: workshops/2025-07-16/walkthrough/01-agent.baml lines 11-27
// License: Apache 2.0

function DetermineNextStep(
    thread: string
) -> DoneForNow {
    client Qwen3              // ← Provider's LLM-client config lives in BAML
    prompt #"                 // ← Multi-line raw string (Go `` ` `` equivalent)
        {{ _.role("system") }}
        You are a helpful assistant that responds to the user's message.

        {{ _.role("user") }}
        You are given the following thread of events:
        {{ thread }}

        What should the next step be?
    "#
}
```

Annotation:

1. `function DetermineNextStep(thread: string) -> DoneForNow`: BAML's
   function declaration. Our Go `Provider.DetermineNextStep(ctx, string)`
   has the same shape.
2. `client Qwen3`: BAML-side LLM client. We move this into the `Provider`
   implementation (`RecordingProvider` here; `OpenAIProvider` in Phase G).
3. `{{ _.role("system") }}`: BAML's role marker. We use the plain string
   `SYSTEM:`. Phase G's real-LLM provider parses these markers back into
   the `messages[].role` JSON for the API call.
4. `{{ thread }}`: BAML's variable substitution. Equivalent to Go's
   `{{ .UserInput }}` (which becomes `{{ .Thread }}` in s03).

## Why we use `text/template` instead of writing our own format

Go's standard library ships a battle-tested template engine. Building
our own would mean:

- writing a lexer + parser for `{{ .X }}` substitutions
- handling escaping
- writing tests for both

…all for zero teaching value. The factor-02 principle is about owning the
**template** — not the engine that renders it. `text/template` is part
of our explicit prompt: a learner can read its docs and know exactly
what `{{ .UserInput }}` does.

## Reading map

- Upstream's complete prompt arc: read in order
  - `content/factor-02-own-your-prompts.md` (the why)
  - `workshops/2025-07-16/walkthrough/01-agent.baml` (the simplest prompt)
  - `workshops/2025-07-16/walkthrough/05-agent.baml` (multi-tool prompt)
  - `workshops/2025-07-16/walkthrough/07-agent.baml` (prompt with `output_format` injection)
- In our curriculum: s02 → s04 (where the prompt starts carrying tool
  schemas) → s07 (where human-contact tools join the union).
