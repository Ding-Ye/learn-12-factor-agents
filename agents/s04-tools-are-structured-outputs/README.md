# s04 · Tools are structured outputs

Maps to **Factor 4**. The LLM stops returning prose — every output is a
typed tool with parameters. `main.go` performs ONE dispatch step (the
loop comes in s05).

## Files
| File | Role |
|---|---|
| `types.go` | `NextStep` + `MathPayload` + intent constants |
| `tools.go` | `Tool` interface + 4 math tools + `Registry` |
| `provider.go` | `ScriptedProvider` (canned routes by input substring) |
| `main.go` | single-step dispatch |
| `tools_test.go` | 7 tests |

## Run
```bash
cd agents/s04-tools-are-structured-outputs
go test ./...
go run . "add 2 and 3"
# → intent=add payload={"a":2,"b":3} result=5
```
