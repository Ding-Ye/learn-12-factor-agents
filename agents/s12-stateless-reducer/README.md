# s12 · Stateless reducer

Maps to **Factor 12**. The whole loop is re-expressed as a pure
`Reduce(Thread, Event) Thread`. No mutable pointers; no global state.
Replay is trivial (same events → same final thread); fork is trivial
(two divergent tails on the same prefix).

## Files
| File | Role |
|---|---|
| `types.go` | NextStep + payloads |
| `events.go` | Event with `json.RawMessage` for byte-equal compares |
| `thread.go` | Value-type Thread; Append copies; Equal byte-compares |
| `reducer.go` | `Reduce` + `ReduceMany` + `IsDone` |
| `loop.go` | RunAgent is a thin shell over Reduce |
| `main.go` | end-to-end demo |
| `reducer_test.go` | 6 tests: replay, fork, no-mutation, auto-step, end-to-end |

## Run
```bash
go test ./...
go run .
```
