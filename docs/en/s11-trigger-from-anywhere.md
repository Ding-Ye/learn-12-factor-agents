---
title: "s11 · Trigger from anywhere (webhook / Slack)"
chapter: 11
slug: s11-trigger-from-anywhere
est_read_min: 8
---

# s11 · Trigger from anywhere (webhook / Slack)

> What you learn here: agents shouldn't only launch from the CLI. A `Trigger` interface ingests Slack messages, HumanLayer webhooks, cron events, etc., and translates them into threads. Downstream looks identical to s06.

---

## Problem / The gap

s06-s10 assumed `POST /thread {message}` was the launch point. Real production entry points are:

- Slack messages (user @-mentions the bot)
- HumanLayer webhooks (human approves → resume agent)
- Cron ticks (every 9am, check-email agent fires)
- Linear webhooks (issue state changes)
- Email forwarding (IMAP)

Hand-writing "parse → seed thread → call RunAgent" per entry point invites copy-paste rot.

Upstream factor-11's answer: **abstract "parse external payload" into a `Trigger` interface**. Each implementation does its own parsing and produces an `Outcome{FreshUserInput?, ResumeThreadID?, HumanResponse?}`. The server reads the Outcome and either spawns or resumes.

## Solution / Mental model

Three decisions:

1. **`Trigger` interface has two methods**: `Source() string` (for trigger-event audit) and `Trigger(r *http.Request) (Outcome, error)`. New trigger = a new struct implementing the two.
2. **`Outcome` is a union value, not two interfaces**: `FreshUserInput` for new threads, `ResumeThreadID + HumanResponse` for resume. `IsFresh()` / `IsResume()` self-check. Value semantics keep tests simple (no mocks).
3. **Routing uses `map[string]Trigger`**: `POST /triggers/{name}` looks up the trigger, calls it, processes the outcome. Registering a trigger = one line in the map.

## How It Works

```
   external source (Slack/HumanLayer/...)
            │
            ▼
   POST /triggers/{name}  ──► Server.handleTrigger
                                       │
                                       ▼
                              Trigger.Trigger(r) → Outcome
                                       │
                            ┌──────────┴──────────┐
                            │                      │
                  outcome.IsFresh()    outcome.IsResume()
                            │                      │
                            ▼                      ▼
                  NewThread(trigger+user) thread = Store.Get(ID)
                  Store.Create             thread.Append(human_response)
                            │                      │
                            └──────────┬──────────┘
                                       ▼
                                  RunAgent
                                       │
                                       ▼
                                  Store.Update + JSON
```

Core 30 lines (excerpts from `triggers/types.go` + `server.go`):

```go
type Outcome struct {
    FreshUserInput string
    ResumeThreadID string
    HumanResponse  string
    Raw            map[string]any
}

type Trigger interface {
    Source() string
    Trigger(r *http.Request) (Outcome, error)
}

// inside server:
outcome, err := trigger.Trigger(r)
switch {
case outcome.IsFresh():
    thread := NewThread(
        NewTriggerEvent(trigger.Source(), outcome.Raw),
        NewUserInputEvent(outcome.FreshUserInput),
    )
    id := s.Store.Create(thread)
    final, _ := RunAgent(r.Context(), thread, s.Provider)
    s.Store.Update(id, final)
case outcome.IsResume():
    thread, _ := s.Store.Get(outcome.ResumeThreadID)
    thread.Append(NewHumanResponseEvent(outcome.HumanResponse))
    final, _ := RunAgent(r.Context(), thread, s.Provider)
    s.Store.Update(outcome.ResumeThreadID, final)
}
```

**Three non-obvious bits:**

1. **`Outcome` uses value semantics, not an interface** — each trigger returns a typed `Outcome`. Making `Outcome` an interface (`type Outcome interface { Apply(*Server) }`) would be more OO but hide the two branches; we'd rather the server's switch makes them visible.
2. **`NewTriggerEvent` enters the thread** as a breadcrumb. The orchestrator/loop never reads it, but it answers "where did this thread come from?" at debug time.
3. **Resume requires an existing `ResumeThreadID`** — missing → 404. Prevents "manually-crafted webhook" attacks from spawning unauthorized threads.

## What Changed vs s10

```diff
+ triggers/ sub-package
+   - types.go: Trigger interface + Outcome
+   - slack.go: SlackTrigger
+   - webhook.go: WebhookTrigger (HumanLayer-style)
+ events.go: + EventTypeTrigger + NewTriggerEvent
- server.go: POST /thread (s06-style)
+ server.go: POST /triggers/{name} (routed by Trigger map)
- 6 tests
+ 5 tests (both trigger paths + error shapes)
```

Semantically: s10 isolated sub-agent contexts; s11 isolates entry points. Both serve the same goal — preventing one big surface area.

## Try It

```bash
cd agents/s11-trigger-from-anywhere

go test -v ./...

go run . :8080 &

# Slack-shaped trigger
curl -s -X POST localhost:8080/triggers/slack \
     -H "Content-Type: application/json" \
     -d '{"event":{"type":"message","text":"add 2 and 3","channel":"C1"}}' | python3 -m json.tool

# Resume via webhook (use thread_id from the previous response)
curl -s -X POST localhost:8080/triggers/webhook \
     -H "Content-Type: application/json" \
     -d '{"event":{"spec":{"state":{"thread_id":"<id>"}},"status":{"response":"ok"}}}' | python3 -m json.tool

kill %1
```

Expected: first trigger returns a thread_id + events (trigger event + user_input + full loop). Second webhook appends `human_response` to the existing thread.

## Upstream Source Reading

```upstream:workshops/2025-07-16/walkthrough/12-server.ts#L31-L99
// Source: workshops/2025-07-16/walkthrough/12-server.ts lines 31-99
// License: Apache 2.0

app.post('/thread', async (req: Request, res: Response) => {
    const body = req.body as V1Beta2EmailEventReceived | { message: string };
    let thread: Thread;

    if ('event' in body) {
        const event = body.event;
        thread = new Thread([{ type: 'email_received', data: event }]);
    } else {
        thread = new Thread([{ type: 'user_input', data: body.message }]);
    }
    // ... store + run agent + return
});

app.post('/webhook', async (req: Request, res: Response) => {
    const response = req.body as V1Beta2HumanContactCompleted;
    const humanResponse: string = response.event.status?.response;
    const threadId = response.event.spec.state?.thread_id;
    const thread = store.get(threadId);
    thread.events.push({ type: 'human_response', data: humanResponse });
    const newThread = await agentLoop(thread);
    store.update(threadId, newThread);
});
```

**Reading notes:**

- **Upstream's `if ('event' in body)` lives inside `/thread`** — duck typing decides. Our Go port lifts the decision into `Trigger.Trigger`. Go's lack of duck typing actually clarifies the boundary.
- **Upstream has separate endpoints** — `/thread` and `/webhook`. We collapse to `/triggers/{name}`. Separate endpoints are more explicit; a single routed endpoint scales better. Five triggers → separate endpoints fine; thirty → use a map.
- **What we omit**: `V1Beta2HumanContactCompleted` is the HumanLayer SDK type. Real implementations verify HMAC signatures, respect retry-after headers, handle idempotency keys. We do none of that; Appendix B exercise #5 covers them.
- **Optional chaining `?.`** — upstream uses TypeScript optional chaining; we get the same effect via struct decoding (zero values for missing fields).
- **Pull-based triggers** (cron, IMAP) aren't in upstream either; they live in user space.

**Want to read more?** `content/factor-11-trigger-from-anywhere.md` is one paragraph that nails the idea; `12-server.ts` is the reference implementation.

---

**Up next, s12:** the loop is refactored from "mutate a Thread pointer" to a pure `Reduce(Thread, Event) Thread`. Same input → same output forever, replay + fork tests become one-liners.
