# s08 · Own your control flow

Maps to **Factor 8**. The s07 loop's implicit branching becomes the
typed `Action` enum: `ActionLoop`, `ActionBreak`, `ActionFinish`,
`ActionEscalate`. `ControlFlow(thread, next, registry) Action` is pure
and individually testable.

## Files
| File | Role |
|---|---|
| `types.go`, `events.go`, `thread.go`, `tools.go` | as before |
| `controlflow.go` | Action enum + ControlFlow + KnownIntents |
| `loop.go` | RunAgent dispatches via ControlFlow |
| `controlflow_test.go` | 7 tests including exhaustiveness |

## Run
```bash
go test ./...
go run .
```
