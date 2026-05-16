# s03 · Own your context window (Thread + Event)

Maps to **Factor 3** of upstream 12-factor agents. The "input" the provider
sees is no longer a raw string — it's a serialized list of `Event` values
that the agent appends to as it runs.

## What we ship

| File | Role |
|---|---|
| `events.go` | `Event` type + 3 constructors |
| `thread.go` | `Thread` (Append, LastEvent, SerializeForLLM) |
| `prompt.go` | template now takes `{{ .Thread }}` not `{{ .UserInput }}` |
| `provider.go` | `EchoThreadProvider` (counts user_input events) |
| `main.go` | seeds thread, renders, calls provider |
| `thread_test.go` | 6 tests |

## Run

```bash
cd agents/s03-own-your-context-window
go test ./...
go run . "hello"
# → intent=done_for_now message="Thread received with 1 user_input event."
```
