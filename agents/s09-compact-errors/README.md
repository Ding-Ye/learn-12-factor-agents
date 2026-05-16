# s09 · Compact errors into context

Maps to **Factor 9**. Tool errors no longer terminate the agent — they
become `error` events on the thread, and the LLM reads them next turn
to self-correct. `ConsecutiveErrors(thread)` derives the retry counter
from the event log; ≥3 → escalate.

## Files
| File | Role |
|---|---|
| `types.go` | + `MaxConsecutiveErrors = 3` |
| `events.go` | + `NewErrorEvent` |
| `thread.go` | + `ConsecutiveErrors(thread)` |
| `tools.go` | + `SafeExecute` (panic recover) |
| `loop.go` | RunAgent appends error events + checks counter |
| `errors_test.go` | 6 tests including panic recovery |

## Run
```bash
go test ./...
go run .
```
