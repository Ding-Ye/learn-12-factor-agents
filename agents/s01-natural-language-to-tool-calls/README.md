# s01 · Minimum agent primitive — Natural language → tool calls

Maps to **Factor 1** of the upstream "12 Factor Agents" guide. This chapter
defines the wire format between the LLM and the agent without yet talking
to a real LLM.

## What we ship

| File | Role |
|---|---|
| `types.go` | `NextStep` (tagged union) + `DoneForNowPayload` |
| `provider.go` | `Provider` interface + `EchoProvider` stub |
| `main.go` | CLI: read argv, call provider, print result |
| `provider_test.go` | 5 tests covering wire format + render fallbacks |

## Run

```bash
cd agents/s01-natural-language-to-tool-calls
go test ./...
go run . "hello"
# → intent=done_for_now message="Hello! How can I assist you today?"
```

## Why this is the bootstrap

Everything from s02 onward extends this `Provider` interface — first by
controlling the prompt (`s02`), then by serializing a thread of events
(`s03`), and finally by returning real tool intents (`s04+`). Pinning the
return type as `NextStep{Intent, Data}` here means later chapters never
need to change the call site.

## Upstream source reading

See `docs/{zh,en}/s01-natural-language-to-tool-calls.md` for the annotated
walkthrough of the upstream files we draw from:

- `content/factor-01-natural-language-to-tool-calls.md` (concepts)
- `workshops/2025-07-16/walkthrough/01-agent.baml` (the BAML DoneForNow
  class + DetermineNextStep function we mirror in `types.go`)
- `workshops/2025-07-16/walkthrough/01-agent.py` (the single-shot call
  site mirrored in `main.go`)
