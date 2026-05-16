# s07 · Contact humans with tools

Maps to **Factor 7**. Two new tools (`RequestApproval`, `AskClarification`)
return `ErrHumanContact` from Execute; the loop catches that and exits
cleanly. The HTTP handler detects `Thread.AwaitingHuman()` and adds a
`response_url` to the reply.

## Files
| File | Role |
|---|---|
| `types.go` | + RequestApprovalPayload, AskClarificationPayload, new intents |
| `events.go` | unchanged from s06 |
| `thread.go` | + AwaitingHuman + IsHumanIntent |
| `tools.go` | + RequestApprovalTool, AskClarificationTool, ErrHumanContact |
| `loop.go` | RunAgent catches ErrHumanContact |
| `store.go` | unchanged from s06 |
| `server.go` | + response_url in ThreadView |
| `main.go` | server with approval-flow demo |
| `server_test.go` | 5 tests |

## Run
```bash
go test ./...
go run . :8080
```
