# s05 · Unify execution state

Maps to **Factor 5**. All state lives in `Thread.Events`. The agent loop
(`RunAgent`) is now multi-step and derives "should I continue?" by
querying the thread (`IsDone`), not by checking a side variable.

## Files
| File | Role |
|---|---|
| `types.go` | NextStep + payload types |
| `events.go` | Event + 3 constructors |
| `thread.go` | Thread + IsDone + LastToolCall |
| `tools.go` | 4 math tools + Registry |
| `provider.go` | ScriptedSequenceProvider (multi-step canned) |
| `loop.go` | RunAgent — the agent loop |
| `main.go` | 3-turn demo |
| `loop_test.go` | 6 tests |

## Run
```bash
cd agents/s05-unify-execution-state
go test ./...
go run .
```
