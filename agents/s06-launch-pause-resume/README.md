# s06 · Launch / pause / resume HTTP API

Maps to **Factor 6**. The s05 agent loop now lives behind a `net/http`
service. Three endpoints: `POST /thread`, `GET /thread/{id}`, `POST
/thread/{id}/response`. In-memory `ThreadStore`.

## Files
| File | Role |
|---|---|
| `types.go` | NextStep + payload types |
| `events.go` | + `human_response` event |
| `thread.go` | Thread |
| `tools.go` | Tool + Registry (2 tools: Add, Multiply) |
| `loop.go` | RunAgent (carried over from s05) |
| `store.go` | ThreadStore interface + MemoryStore |
| `server.go` | Server + 3 handlers |
| `main.go` | spin up :8080 |
| `server_test.go` | 6 tests via httptest |

## Run
```bash
cd agents/s06-launch-pause-resume
go test ./...
go run . &
curl -s -X POST localhost:8080/thread -d '{"message":"hi"}' | jq
```
