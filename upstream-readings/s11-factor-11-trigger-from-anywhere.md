# Upstream reading — Factor 11: Trigger from anywhere

> Source: humanlayer/12-factor-agents @ d20c728.

## Source: `workshops/2025-07-16/walkthrough/12-server.ts` (31-99)

```typescript
// License: Apache 2.0
app.post('/thread', async (req, res) => {
    const body = req.body as V1Beta2EmailEventReceived | { message: string };
    let thread: Thread;
    if ('event' in body) {
        thread = new Thread([{ type: 'email_received', data: body.event }]);
    } else {
        thread = new Thread([{ type: 'user_input', data: body.message }]);
    }
    // store, run, return
});

app.post('/webhook', async (req, res) => {
    const response = req.body as V1Beta2HumanContactCompleted;
    const humanResponse: string = response.event.status?.response;
    const threadId = response.event.spec.state?.thread_id;
    const thread = store.get(threadId);
    thread.events.push({ type: 'human_response', data: humanResponse });
    await agentLoop(thread);
});
```

## Mapping

| Upstream | Our Go | File |
|---|---|---|
| `if ('event' in body)` duck typing | `Trigger.Trigger` interface decision | triggers/types.go |
| Separate `/thread` + `/webhook` | One `POST /triggers/{name}` | server.go |
| `V1Beta2*` SDK types | hand-rolled `slackBody`, `webhookBody` structs | triggers/*.go |
| HMAC, retry-after, idempotency | (not implemented — Appendix B #5) | — |

## Reading map

- s11 → s12: triggers continue to feed the same `Reduce(Thread, Event)
  → Thread` pipeline; trigger events are just one more kind of Event.
- s11 extensions in Appendix B:
  - real Slack OAuth + HMAC signature verification
  - cron-style pull triggers (re-poll every N minutes)
  - email forwarder (IMAP → trigger).
