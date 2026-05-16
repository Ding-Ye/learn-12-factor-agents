---
title: "s07 · Contact humans with tools"
chapter: 7
slug: s07-contact-humans-with-tools
est_read_min: 9
---

# s07 · Contact humans with tools

> What you learn here: humans aren't a special channel — they're another kind of "tool." When the LLM emits `request_approval` or `request_more_information`, the loop **doesn't call Execute** and exits cleanly. The HTTP layer adds a `response_url` so the next `POST /response` resumes.

---

## Problem / The gap

s06 plumbed start / inspect / resume over HTTP, but `human_response` events still arrived only via explicit client POSTs — the agent never decided to "pause and wait for a human." Real flows want the LLM to say "this $100k transfer needs approval" on its own.

Upstream factor-07's answer: **humans are tools**. The LLM's schema includes `RequestApproval` / `AskClarification`; when it emits one, code-side **doesn't execute** — it pauses the thread for a human.

## Solution / Mental model

Three decisions:

1. **`ErrHumanContact` sentinel error** — defined in `tools.go`. `RequestApprovalTool.Execute` and `AskClarificationTool.Execute` return it. `RunAgent` checks via `errors.Is(err, ErrHumanContact)` and exits cleanly (not an error).
2. **`Thread.AwaitingHuman()` is the predicate** — derived from the thread's last event (a human-intent tool_call). Same shape as s05's `IsDone`.
3. **`ThreadView` adds `response_url`** — when awaiting, the server fills in `BaseURL + /thread/{id}/response`. Clients don't need to know routing.

## How It Works

```
POST /thread {message} ──► handleCreate ──► RunAgent ──┐
                                                        │
                                       provider returns approvalStep
                                                        │
                                       loop appends tool_call
                                                        │
                                       Registry.Lookup → RequestApprovalTool
                                                        │
                                       tool.Execute → ErrHumanContact
                                                        │
                                       errors.Is(err, ErrHumanContact) → return (no error)
                                                        │
                                                        ▼
                                       Store.Update + view{Awaiting:true, response_url}

POST /thread/{id}/response {message} ──► appends human_response ──► RunAgent (resumes)
```

Core 30 lines (excerpt from `loop.go` + `tools.go`):

```go
var ErrHumanContact = errors.New("human contact requested")

type RequestApprovalTool struct{}
func (RequestApprovalTool) Intent() string { return IntentRequestApproval }
func (RequestApprovalTool) Execute(_ context.Context, _ json.RawMessage) (any, error) {
    return nil, ErrHumanContact
}

// inside RunAgent:
result, err := tool.Execute(ctx, next.Data)
if err != nil {
    if errors.Is(err, ErrHumanContact) {
        // No tool_response — handler will write response_url
        return thread, nil
    }
    return thread, fmt.Errorf("tool %q at step %d: %w", next.Intent, step, err)
}
thread.Append(NewToolResponseEvent(result))
```

**Three non-obvious bits:**

1. **`errors.Is` instead of type assertion** — future wrappers (`fmt.Errorf("...%w", ErrHumanContact)`) still match with `errors.Is`. Type assertion would false-negative.
2. **Human-contact tool doesn't append `tool_response`** — mirrors upstream `12-server.ts:31-61`. The response will come once the human replies. Otherwise the LLM sees `tool_call(approval)` immediately followed by `tool_response(nil)`, which confuses it.
3. **`AwaitingHuman` lives in `thread.go`, not `server.go`** — the predicate belongs to the data structure. s10 and s12 reuse it; the server is just one caller.

## What Changed vs s06

```diff
+ types.go: + RequestApprovalPayload, AskClarificationPayload, 2 new intents
+ thread.go: + AwaitingHuman + IsHumanIntent
+ tools.go: + RequestApprovalTool, AskClarificationTool, ErrHumanContact sentinel
+ loop.go: errors.Is(err, ErrHumanContact) clean exit
+ server.go: ThreadView includes Awaiting + ResponseURL; view fills automatically
- 6 tests (s06)
+ 5 tests (AwaitingHuman, loop early exit, HTTP round-trip)
```

Semantically: s06's loop exits on `done_for_now` or error; s07 adds a third path — `human contact`. All three are clean exits (no error); `AwaitingHuman()` distinguishes them.

## Try It

```bash
cd agents/s07-contact-humans-with-tools

go test -v -race ./...

go run . :8080 &

curl -s -X POST localhost:8080/thread \
     -H "Content-Type: application/json" \
     -d '{"message":"send 100k to acme"}' | python3 -m json.tool
# Response includes "awaiting": true and "response_url": ".../thread/<id>/response"

# Resume via response_url
curl -s -X POST 'http://localhost:8080/thread/<id>/response' \
     -H "Content-Type: application/json" \
     -d '{"message":"approved"}' | python3 -m json.tool
# Awaiting disappears after resume

kill %1
```

Expected: first POST yields thread.events = user_input → tool_call(multiply 1000×100) → tool_response(100000) → tool_call(request_approval) → (no tool_response). After resume, human_response + tool_call(done_for_now).

## Upstream Source Reading

```upstream:workshops/2025-07-16/walkthrough/05-agent.baml#L29-L37
// Source: workshops/2025-07-16/walkthrough/05-agent.baml lines 29-37
// License: Apache 2.0

class ClarificationRequest {
  intent "request_more_information"
  message string
}

function DetermineNextStep(thread: string)
    -> DoneForNow | AddTool | SubtractTool | MultiplyTool | DivideTool | ClarificationRequest {
    client Qwen3
    prompt #" ... "#
}
```

```upstream:workshops/2025-07-16/walkthrough/05-agent.py#L24-L40
# Source: workshops/2025-07-16/walkthrough/05-agent.py lines 24-40
# License: Apache 2.0

def agent_loop(thread: Thread) -> AgentResponse:
    b = get_baml_client()
    while True:
        next_step = b.DetermineNextStep(thread.serialize_for_llm())
        thread.events.append({"type": next_step.intent, "data": next_step})

        if next_step.intent == "done_for_now":
            return next_step
        elif next_step.intent == "request_more_information":
            clarification = clarification_handler(next_step.message)
            thread.events.append({"type": "clarification_response", "data": clarification})
        # ... math tool branches
```

**Reading notes:**

- **`ClarificationRequest` joins the union return type** — upstream adds the class to BAML's return-type union; the LLM learns it's a legal emission. We have no codegen, so the prompt's schema text carries this info (hand-written in Phase G).
- **Inline handling vs sentinel error** — upstream branches inline on `request_more_information` and calls `clarification_handler`; we raise `ErrHumanContact` for the loop to handle. Trade-off: upstream blocks synchronously; we yield to HTTP.
- **`clarification_response` event vs our `human_response`** — upstream uses an intent-specific event type; we unify on `human_response`. Simpler, slightly less typed structure.
- **`request_approval` isn't in upstream BAML** — only `request_more_information` lives there. We add `request_approval` (from `12-server.ts`'s human-approval flow). Pedagogically, approval makes "breaking the loop" more visible.
- **`response_url` is a HumanLayer SDK convention** — `09-server.ts:23-26` writes it; we mirror.

**Want to read more?** `content/factor-07-contact-humans-with-tools.md:21-46` plus `workshops/2025-07-16/walkthrough/12-server.ts:31-99` (webhook resume flow). s11 picks up the `12-server.ts` thread.

---

**Up next, s08:** the "if intent in [...] / else if ... / else if ..." chain inside the loop becomes a typed `ControlFlow(thread, next) Action` function. Each branch returns `ActionLoop` / `ActionBreak` / `ActionEscalate`. Adding a new intent triggers a compile error, forcing the developer to make the dispatch explicit.
