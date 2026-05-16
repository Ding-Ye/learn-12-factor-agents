# s02 · Own your prompts

Maps to **Factor 2** of upstream 12-factor agents. This chapter pulls the
prompt out of the framework's hidden corners and writes it as an explicit
Go `text/template`.

## What we ship

| File | Role |
|---|---|
| `types.go` | `NextStep`, `DoneForNowPayload` (carried over from s01) |
| `prompt.go` | `RenderPrompt` + `PromptHash` |
| `provider.go` | `RecordingProvider` (echoes prompt hash back) |
| `main.go` | CLI wiring |
| `prompt_test.go` | 6 tests covering template + hash + roundtrip |

## Run

```bash
cd agents/s02-own-your-prompts
go test ./...
go run . "hello"
# → intent=done_for_now message="Acknowledged. prompt_hash=<16-hex-chars>"
```

The `prompt_hash` in the output proves the rendered prompt actually
reached the provider — change the template, the hash changes.

## Upstream source reading

See `docs/{zh,en}/s02-own-your-prompts.md` for the side-by-side with
upstream `01-agent.baml:12-27` (the BAML prompt block).
