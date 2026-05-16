# Upstream reading — Factor 6: Launch / Pause / Resume

> Source: humanlayer/12-factor-agents @ d20c728.

## Source A: `workshops/2025-07-16/walkthrough/09-server.ts` (1-60)

```typescript
// License: Apache 2.0
import express from 'express';
import { Thread, agentLoop } from './agent';
import { ThreadStore } from './state';

const app = express();
app.use(express.json());
const store = new ThreadStore();

app.post('/thread', async (req, res) => {
    const thread = new Thread([{ type: 'user_input', data: req.body.message }]);
    const threadId = store.create(thread);
    const newThread = await agentLoop(thread);
    store.update(threadId, newThread);
    res.json({ thread_id: threadId, ...newThread });
});

app.get('/thread/:id', (req, res) => { ... });

app.post('/thread/:id/response', async (req, res) => {
    const thread = store.get(req.params.id);
    if (!thread) return res.status(404).json({error: 'not found'});
    thread.events.push({ type: 'human_response', data: req.body.message });
    const newThread = await agentLoop(thread);
    store.update(req.params.id, newThread);
    res.json({ thread_id: req.params.id, ...newThread });
});
```

## Source B: `workshops/2025-07-16/walkthrough/09-state.ts` (1-23)

```typescript
// License: Apache 2.0
import { Thread } from './agent';

export class ThreadStore {
    private threads: Map<string, Thread> = new Map();
    create(thread: Thread): string { const id = crypto.randomUUID(); this.threads.set(id, thread); return id; }
    get(id: string): Thread | undefined { return this.threads.get(id); }
    update(id: string, thread: Thread): void { this.threads.set(id, thread); }
}
```

## Mapping

| Upstream | Our Go | File |
|---|---|---|
| express app | `http.ServeMux` (Go 1.22 routing) | `server.go` |
| `crypto.randomUUID()` | `crypto/rand` → 12-byte hex | `store.go` |
| `Map<string, Thread>` | `map[string]*Thread` + `sync.Mutex` | `store.go` |
| `async/await` | synchronous handler + `r.Context()` | `server.go` |
| `thread.events.push` | `thread.Append` | `thread.go` |

## Reading map

- s06 → s07: human-contact tools are detected in `RunAgent`; `AwaitingHuman`
  signals the server to return a `response_url`.
- s06 → s11: `/triggers/{name}` mounts pluggable trigger handlers that
  produce threads from Slack/email payloads instead of plain JSON.
- s06 extension: swap `MemoryStore` for a `SQLiteStore` to persist
  across process restarts.
