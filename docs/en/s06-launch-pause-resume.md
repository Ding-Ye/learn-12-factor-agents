---
title: "s06 · Launch / pause / resume HTTP API"
chapter: 6
slug: s06-launch-pause-resume
est_read_min: 10
---

# s06 · Launch / pause / resume HTTP API

> What you learn here: host the s05 agent loop behind a tiny HTTP service. Three endpoints — `POST /thread` to launch, `GET /thread/{id}` to inspect, `POST /thread/{id}/response` to resume.

---

## Problem / The gap

s05 made the loop work multi-step, but only inside a single CLI invocation. Real flows look different: user fires a request via Slack, agent runs halfway and needs approval, the human comes back 30 minutes later, the agent resumes. That demands (1) persistent threads, (2) HTTP endpoints to launch / inspect / resume, (3) concurrent requests that don't step on each other.

Upstream factor-06 prescribes exactly this thin HTTP wrapper. In s06 we build it with Go's stdlib: `net/http`, a `sync.Mutex`-guarded `map[string]*Thread`, and `crypto/rand` for thread IDs.

## Solution / Mental model

Three decisions:

1. **`ThreadStore` is an interface**: the in-memory `MemoryStore` is the only implementation; swap in sqlite/redis by writing a new type. Extension exercise in Appendix B.
2. **`RunAgent` runs synchronously inside the HTTP handler**: no goroutine spawn, no `{status:"processing"}` response. Synchronous keeps the teaching narrative simple (request end = thread terminal). The async trade-off shows up in s11 commentary.
3. **`POST /response` reuses `RunAgent`**: append a `human_response` event → call `RunAgent` again. s07 will let the agent actually decide whether to "skip the next LLM call" based on what's in the thread.

## How It Works

```
   client ─POST /thread {message}─► Server.handleCreate
                                          │
                                          ▼
                                  NewThread + Store.Create(t)
                                          │
                                          ▼
                                  RunAgent(ctx, t, provider, registry)
                                          │
                                          ▼
                                  Store.Update + JSON response

   client ─GET /thread/{id}──► Server.handleGet ──► Store.Get ──► JSON

   client ─POST /thread/{id}/response {message}─► Server.handleResponse
                                                       │
                                                       ▼
                                               Store.Get(id)
                                                       │
                                                       ▼
                                       thread.Append(NewHumanResponseEvent)
                                                       │
                                                       ▼
                                       RunAgent(ctx, thread, ...)
                                                       │
                                                       ▼
                                                  JSON response
```

Core 30 lines (excerpt from [`agents/s06-launch-pause-resume/server.go`](https://github.com/Ding-Ye/learn-12-factor-agents/blob/main/agents/s06-launch-pause-resume/server.go)):

```go
func (s *Server) Handler() http.Handler {
    mux := http.NewServeMux()
    mux.HandleFunc("POST /thread", s.handleCreate)
    mux.HandleFunc("GET /thread/{id}", s.handleGet)
    mux.HandleFunc("POST /thread/{id}/response", s.handleResponse)
    return mux
}

func (s *Server) handleCreate(w http.ResponseWriter, r *http.Request) {
    var req CreateRequest
    if err := json.NewDecoder(r.Body).Decode(&req); err != nil { /* 400 */ }
    thread := NewThread(NewUserInputEvent(req.Message))
    id := s.Store.Create(thread)
    final, err := RunAgent(r.Context(), thread, s.Provider, s.Registry)
    if err != nil { /* 500 + persist partial */ }
    s.Store.Update(id, final)
    writeJSON(w, http.StatusOK, ThreadView{ID: id, Thread: final})
}
```

**Three non-obvious bits:**

1. **`http.NewServeMux` with Go 1.22+ routing** — `"POST /thread"` and `r.PathValue("id")` give you REST routing without gorilla/mux. This is the standard library doing more than it used to.
2. **`Store.Update` on error too** — if the agent crashes mid-flight we still persist the partial thread so a debugger can `GET` and inspect the last event. Losing the trace would be a worse bug than the crash itself.
3. **`MemoryStore` uses `sync.Mutex`, not `sync.RWMutex`** — writes dominate (every `RunAgent` calls `Update`), so the RWMutex read optimization yields little. Plain `Mutex` is also easier to reason about in a teaching context.

## What Changed vs s05

```diff
+ store.go (new — ThreadStore + MemoryStore)
+ server.go (new — Server + 3 handlers)
+ events.go: + EventTypeHumanResponse + NewHumanResponseEvent
- main.go: CLI demo
+ main.go: HTTP server on :8080
- 6 tests (loop_test.go)
+ 6 tests (server_test.go via httptest, including concurrent race test)
```

Semantically: s05's "loop" was a CLI lifecycle; s06's is split across HTTP requests. The Thread survives past the original request.

## Try It

```bash
cd agents/s06-launch-pause-resume

go test -v -race ./...

# start server
go run . :8080 &

# launch a thread
curl -s -X POST localhost:8080/thread \
     -H "Content-Type: application/json" \
     -d '{"message":"go"}' | python3 -m json.tool

# inspect by ID (use the thread_id from the previous response)
curl -s localhost:8080/thread/<id> | python3 -m json.tool

# resume (appends human_response, re-runs RunAgent)
curl -s -X POST localhost:8080/thread/<id>/response \
     -H "Content-Type: application/json" \
     -d '{"message":"please continue"}' | python3 -m json.tool

# stop the server
kill %1
```

Expected: thread_id is a 24-char hex string; thread.events contains user_input + 3 tool_call + 2 tool_response. After resume, one more `human_response` event.

## Upstream Source Reading

```upstream:workshops/2025-07-16/walkthrough/09-server.ts#L1-L60
// Source: workshops/2025-07-16/walkthrough/09-server.ts lines 1-60
// License: Apache 2.0

import express from 'express';
import { Thread, agentLoop } from './agent';
import { ThreadStore } from './state';

const app = express();
app.use(express.json());

const store = new ThreadStore();

app.post('/thread', async (req, res) => {
    const thread = new Thread([{
        type: 'user_input',
        data: req.body.message
    }]);
    const threadId = store.create(thread);

    const newThread = await agentLoop(thread);
    store.update(threadId, newThread);

    res.json({ thread_id: threadId, ...newThread });
});

app.get('/thread/:id', (req, res) => {
    const thread = store.get(req.params.id);
    if (!thread) { return res.status(404).json({error: 'not found'}); }
    res.json({ thread_id: req.params.id, ...thread });
});

app.post('/thread/:id/response', async (req, res) => {
    const thread = store.get(req.params.id);
    if (!thread) { return res.status(404).json({error: 'not found'}); }
    thread.events.push({ type: 'human_response', data: req.body.message });
    const newThread = await agentLoop(thread);
    store.update(req.params.id, newThread);
    res.json({ thread_id: req.params.id, ...newThread });
});
```

```upstream:workshops/2025-07-16/walkthrough/09-state.ts#L1-L23
// Source: workshops/2025-07-16/walkthrough/09-state.ts lines 1-23
// License: Apache 2.0

import { Thread } from './agent';

export class ThreadStore {
    private threads: Map<string, Thread> = new Map();

    create(thread: Thread): string {
        const id = crypto.randomUUID();
        this.threads.set(id, thread);
        return id;
    }

    get(id: string): Thread | undefined {
        return this.threads.get(id);
    }

    update(id: string, thread: Thread): void {
        this.threads.set(id, thread);
    }
}
```

**Reading notes:**

- **Express vs net/http**: upstream uses express; we stay on stdlib `http.ServeMux`. Go 1.22+ supports method + path + path params natively, so no gorilla/mux.
- **`crypto.randomUUID()` vs `crypto/rand`**: upstream uses UUID v4 (36 chars); we use 12 random bytes hex-encoded (24 chars). Functionally equivalent for a single-process store.
- **`async/await` vs Go context**: upstream's handlers are `async`; ours are synchronous and propagate cancellation through `r.Context()`. Go's blocking I/O model maps the two cleanly without explicit await.
- **`thread.events.push` vs `thread.Append`**: upstream mutates the array directly; we wrap it in a method. Equivalent.
- **What we omit**: upstream `12-server.ts:38-58` demonstrates an async mode — `POST /thread` returns `{status:"processing"}` immediately and the agent runs in the background. We keep s06 synchronous; s11 commentary discusses the async trade-off.

**Want to read more?** Upstream `10-server.ts` (human-approval flow) and `12-server.ts` (webhook triggers). We pick those up in s07 and s11 respectively.

---

**Up next, s07:** the agent loop learns to "see" humans. `RequestApproval` and `AskClarification` join the `Tool` interface; when the LLM emits one of those intents, the loop **does not** call `tool.Execute` and returns the thread instead. The HTTP handler detects `AwaitingHuman` and writes a `response_url` into the reply; the next `POST /response` resumes.
