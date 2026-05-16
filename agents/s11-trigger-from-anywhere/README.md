# s11 · Trigger from anywhere

Maps to **Factor 11**. Agents launch from any external source: Slack
message, HumanLayer webhook, cron, etc. Each `Trigger` parses the
request and produces an `Outcome` that the server turns into either a
fresh thread or a resume-existing-thread call.

## Files
| File | Role |
|---|---|
| `types.go`, `events.go`, `thread.go` | + `EventTypeTrigger` |
| `triggers/types.go` | `Trigger` interface + `Outcome` |
| `triggers/slack.go` | SlackTrigger (parses `event.message.text`) |
| `triggers/webhook.go` | WebhookTrigger (HumanLayer-style) |
| `store.go`, `loop.go` | minimal carry-over |
| `server.go` | `POST /triggers/{name}` + `GET /thread/{id}` |
| `server_test.go` | 5 tests |

## Run
```bash
go test ./...
go run . :8080
```
