# s10 · Small, focused agents

Maps to **Factor 10**. An `Orchestrator` composes two sub-agents
(`CalcAgent`, `SummaryAgent`). Each sub-agent has its own thread; the
orchestrator's thread records only sub-agent call/done boundaries.

## Files
| File | Role |
|---|---|
| `types.go`, `events.go`, `thread.go` | + sub-agent event types |
| `orchestrator.go` | Orchestrate function |
| `subagents/calc.go` | CalcAgent (chained math) |
| `subagents/summary.go` | SummaryAgent (templated wrap-up) |
| `main.go` | demo |
| `orchestrator_test.go` | 6 tests |

## Run
```bash
go test ./...
go run .
```
