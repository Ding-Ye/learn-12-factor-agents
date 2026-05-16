---
title: "s_full · End-to-end integration: the twelve factors as one run"
chapter: full
slug: s_full-integration
est_read_min: 14
---

# s_full · End-to-end integration: the twelve factors as one run

> This chapter writes no new code. It threads all twelve factors through one real query so you can see where every line lives and which factor it serves.

---

## Full-stack architecture

```
                                    ┌─────────────────────────────┐
                                    │   external triggers (s11)   │
                                    │   slack / webhook / cron    │
                                    └──────────────┬──────────────┘
                                                   │ Outcome
                                                   ▼
                              ┌─────────────────────────────────────┐
   POST /thread (s06)         │           HTTP Server               │
   POST /response (s06)  ────►│  handleCreate / handleResponse      │
   POST /triggers/{n} (s11)   │  (sync — RunAgent in same goroutine)│
                              └──────────────┬──────────────────────┘
                                             │
                                             ▼
                              ┌─────────────────────────────────────┐
                              │            RunAgent (s05/s12)       │
                              │   ┌─────────────────────────────┐   │
                              │   │ for step < MaxSteps {      │   │
                              │   │   next = Provider.Next(thr) │   │  ← Provider abstracted s01
                              │   │   action = ControlFlow(...) │   │  ← typed Action enum s08
                              │   │   switch action {           │   │
                              │   │     Finish → return         │   │
                              │   │     Break  → return (s07)   │   │  ← human contact
                              │   │     Loop   → execute tool   │   │
                              │   │     Escalate → error event  │   │  ← s09 self-heal
                              │   │   }                         │   │
                              │   │ }                           │   │
                              │   └─────────────────────────────┘   │
                              └──┬──────┬──────────┬────────────────┘
                                 │      │          │
                                 ▼      ▼          ▼
                            ┌────────┐ ┌────┐ ┌─────────────┐
                            │ Thread │ │Tool│ │ Reduce      │
                            │  s03   │ │ s04│ │ (s12, pure) │
                            │  event │ │ Reg│ │             │
                            │  log   │ │istry│ │ Thread →   │
                            └────────┘ └────┘ │   Thread    │
                                              └─────────────┘

                              ┌─────────────────────────────────────┐
                              │       ThreadStore (s06)             │
                              │   in-memory map[string]*Thread      │
                              │   sync.Mutex                        │
                              └─────────────────────────────────────┘

                              Sub-agent composition (s10):
                                Orchestrator → CalcAgent → SummaryAgent
                                each with its own Thread, recorded via
                                subagent_call / subagent_done events.

                              Own your prompt (s02):
                                text/template renders SYSTEM/USER sections
                                with the serialized Thread.
```

Where each factor lives:

| Factor | Code location | Runtime appearance |
|---|---|---|
| 1 NL→tool | `Provider.DetermineNextStep` interface | every LLM call |
| 2 Own your prompts | `RenderPrompt(...)` + `text/template` | before every Provider call |
| 3 Own context window | `Thread{Events []Event}` | seed of every loop |
| 4 Structured tools | `Tool` interface + `Registry` | every dispatch |
| 5 Unified state | `Thread.Events` is the only state | always |
| 6 Launch/pause/resume | `Server` + `ThreadStore` | HTTP entry |
| 7 Human as tool | `RequestApproval` / `AskClarification` tools | `Action == Break` path |
| 8 Own control flow | `Action` enum + `ControlFlow` | the loop's switch |
| 9 Compact errors | `NewErrorEvent` + `ConsecutiveErrors` | tool failure path |
| 10 Small focused agents | `Orchestrator` + `subagents/` | when tasks compose |
| 11 Trigger from anywhere | `Trigger` interface + `triggers/` | external entry points |
| 12 Stateless reducer | `Reduce(Thread, Event) Thread` | s12 onward |

---

## 16-step trace: `"add 5 and 3, then multiply by 2"` + one human approval

| Step | Who does what | File:line (chapter) | Factors touched |
|---|---|---|---|
| 1 | Client `POST /thread {message:"..."}` | `s06/server.go:handleCreate` | 6 |
| 2 | Server builds Thread seeded with `user_input` | `s03/events.go:NewUserInputEvent` | 3, 5 |
| 3 | `Store.Create` returns 12-byte hex thread_id | `s06/store.go:MemoryStore.Create` | 6 |
| 4 | `RunAgent` enters the loop, first provider call | `s05/loop.go:RunAgent` | 5, 8 |
| 5 | Provider receives `Thread.SerializeForLLM()` JSON | `s03/thread.go:SerializeForLLM` | 3 |
| 6 | LLM/stub returns `NextStep{Intent:"add", Data:{a:5,b:3}}` | `s01/types.go:NextStep` | 1, 4 |
| 7 | `ControlFlow` returns `ActionLoop` (math intent in registry) | `s08/controlflow.go` | 8 |
| 8 | Loop appends `tool_call` event, runs `AddTool.Execute` → 8 | `s04/tools.go:AddTool` | 4, 5 |
| 9 | Loop appends `tool_response` event, continues | `s05/events.go:NewToolResponseEvent` | 5 |
| 10 | Second provider call: `NextStep{Intent:"request_approval", Data:{question:"approve multiply?"}}` | `s07/types.go:RequestApprovalPayload` | 7 |
| 11 | `ControlFlow` returns `ActionBreak` (human intent) | `s08/controlflow.go` | 7, 8 |
| 12 | `RunAgent` returns the thread (tool_call without paired tool_response) | `s07/loop.go` | 7 |
| 13 | Server sees `AwaitingHuman()`, response contains `response_url` | `s07/server.go:view` | 7 |
| 14 | 30 minutes later, HumanLayer fires webhook → `POST /triggers/webhook` | `s11/triggers/webhook.go` | 11 |
| 15 | `WebhookTrigger` extracts `ResumeThreadID` + `HumanResponse: "approved"` | `s11/triggers/webhook.go:Trigger` | 11 |
| 16 | Server loads stored thread, appends `human_response`, calls `RunAgent` again | `s11/server.go:handleTrigger` | 6, 11 |

Beyond step 16: the provider's third call returns `multiply` → tool runs (tool_response = 16) → fourth call returns `done_for_now` → loop exits.

**The s12 reducer view of the same trace**: every step is `t = Reduce(t, e)`. The 16 events can be replayed by folding `Reduce` over them again — the final Thread is byte-identical.

---

## Deliberate omissions (what the teaching repo intentionally skips)

| Upstream has | We don't | Why |
|---|---|---|
| BAML codegen | Hand-written `NextStep{Intent, Data json.RawMessage}` + JSON schema strings | No Go-native BAML; codegen is a separate topic |
| Real Anthropic / OpenAI provider | Stub `ScriptedSequenceProvider` | Tests stay hermetic; Phase G adds the real path |
| Durable `ThreadStore` | in-memory `map[string]*Thread` | sqlite/redis is infrastructure; Appendix B exercise |
| HumanLayer SDK integration | Hand-written `WebhookTrigger` | SDK ergonomics would obscure the trigger abstraction |
| Slack OAuth + HMAC verification | Accepts any POST body | Security is a deep topic; exercise in Appendix B |
| Streaming token responses | Plain JSON | SSE/WebSocket is out of scope |
| Async loop (`POST /thread` returns `processing`) | Synchronous in-handler | Sync is linear; matches teaching narrative |
| OpenTelemetry tracing | None | Appendix B exercise |
| BAML's `output_format` schema injection | Hand-written in prompts | Phase G |
| Pre-fetch / appendix-13 optimization | None | Upstream is only markdown |

---

## What you can do now

Having read this chapter you can:

1. **Re-read each chapter's "What Changed vs prev" section** — chained together, they are the diff history of this 16-step trace.
2. **Find any factor's code** via the table above ("Code location" column).
3. **Add a tool** (e.g., `FetchURL`): write a struct + implement `Tool` + register in `DefaultRegistry()` + add the intent to `KnownIntents()`. `ControlFlow` will route `ActionLoop` automatically.
4. **Add a trigger** (e.g., Linear webhook): write a struct + implement `Trigger` + register in the server's `Triggers` map.
5. **Swap the Provider**: from `ScriptedSequenceProvider` to `OpenAIProvider` is a one-line constructor change (with Phase G's `OpenAIProvider`).

Appendix A explores why this style beats framework wrappers. Appendix B gives a full reading path through the upstream repo.
