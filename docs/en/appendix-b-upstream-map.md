---
title: "Appendix B · Upstream reading map"
slug: appendix-b-upstream-map
est_read_min: 10
---

# Appendix B · Upstream reading map

> After 12 chapters + `s_full` + Appendix A, you've rebuilt the 12-factor patterns in Go. This appendix tells you what to read in the upstream repo, in what order.

---

## Recommended reading order

### Step 1 — The manifesto

1. [`README.md`](https://github.com/humanlayer/12-factor-agents/blob/main/README.md) — the "why." Read the first 60 lines (agent-framework critique) and the 12-factor list.
2. [`content/brief-history-of-software.md`](https://github.com/humanlayer/12-factor-agents/blob/main/content/brief-history-of-software.md) — software-design history and where agents fit.

### Step 2 — The twelve principles in detail

Read each factor markdown in order (100-300 lines each):

| Factor | File | Focus lines |
|---|---|---|
| 1 | `content/factor-01-natural-language-to-tool-calls.md` | full |
| 2 | `content/factor-02-own-your-prompts.md` | 14-91 |
| 3 | `content/factor-03-own-your-context-window.md` | 69-139 (serialization trade-offs) |
| 4 | `content/factor-04-tools-are-structured-outputs.md` | 11-50 |
| 5 | `content/factor-05-unify-execution-state.md` | full (short) |
| 6 | `content/factor-06-launch-pause-resume.md` | full |
| 7 | `content/factor-07-contact-humans-with-tools.md` | 21-46 |
| 8 | `content/factor-08-own-your-control-flow.md` | 27-68 |
| 9 | `content/factor-09-compact-errors.md` | 10-59 |
| 10 | `content/factor-10-small-focused-agents.md` | full |
| 11 | `content/factor-11-trigger-from-anywhere.md` | full (short) |
| 12 | `content/factor-12-stateless-reducer.md` | full (12 lines!) |

### Step 3 — TypeScript walkthrough (the main reference implementation)

`workshops/2025-05-17/walkthrough/` is the BAML + TypeScript 12-step build. Read:

1. [`00-package.json`](https://github.com/humanlayer/12-factor-agents/blob/main/workshops/2025-05-17/walkthrough/00-package.json) — dependencies
2. [`01-agent.baml`](https://github.com/humanlayer/12-factor-agents/blob/main/workshops/2025-05-17/walkthrough/01-agent.baml) + [`01-agent.ts`](https://github.com/humanlayer/12-factor-agents/blob/main/workshops/2025-05-17/walkthrough/01-agent.ts) — minimum agent (our s01-s02)
3. [`05-agent.baml`](https://github.com/humanlayer/12-factor-agents/blob/main/workshops/2025-05-17/walkthrough/05-agent.baml) — ClarificationRequest (our s07)
4. [`09-server.ts`](https://github.com/humanlayer/12-factor-agents/blob/main/workshops/2025-05-17/walkthrough/09-server.ts) + [`09-state.ts`](https://github.com/humanlayer/12-factor-agents/blob/main/workshops/2025-05-17/walkthrough/09-state.ts) — HTTP server + ThreadStore (our s06)
5. [`12-server.ts`](https://github.com/humanlayer/12-factor-agents/blob/main/workshops/2025-05-17/walkthrough/12-server.ts) — webhook + async loop (our s11)

### Step 4 — Python walkthrough

`workshops/2025-07-16/walkthrough/` is the Python adaptation. More stable than the TS version, notebook-friendly. Read `01-agent.py` → `03-agent.py` → `05-agent.py` → `07-agent.py`.

### Step 5 — Production template

[`packages/create-12-factor-agent/template/src/`](https://github.com/humanlayer/12-factor-agents/tree/main/packages/create-12-factor-agent/template/src) is the npm-scaffolded "production-ish layout":

- `agent.ts` — lines 89-114 contain the most complete upstream `agentLoop`. Source of our s12 `Reduce` refactor.
- `state.ts` — production ThreadStore (still in-memory)
- `cli.ts` / `server.ts` — entry points

---

## Upstream file → our chapter map

| Upstream file | Our chapter |
|---|---|
| `content/factor-01-natural-language-to-tool-calls.md` | s01 |
| `content/factor-02-own-your-prompts.md` | s02 |
| `workshops/2025-07-16/walkthrough/01-agent.baml` | s01, s02 |
| `workshops/2025-07-16/walkthrough/01-agent.py` | s03 |
| `content/factor-03-own-your-context-window.md` | s03 |
| `content/factor-04-tools-are-structured-outputs.md` | s04 |
| `workshops/2025-07-16/walkthrough/05-agent.baml` | s04, s07 |
| `content/factor-05-unify-execution-state.md` | s05 |
| `workshops/2025-07-16/walkthrough/03-agent.py` | s05, s09 |
| `content/factor-06-launch-pause-resume.md` | s06 |
| `workshops/2025-07-16/walkthrough/09-server.ts` | s06 |
| `workshops/2025-07-16/walkthrough/09-state.ts` | s06 |
| `content/factor-07-contact-humans-with-tools.md` | s07 |
| `workshops/2025-07-16/walkthrough/05-agent.py` | s07 |
| `content/factor-08-own-your-control-flow.md` | s08 |
| `workshops/2025-07-16/walkthrough/07-agent.py` | s08 |
| `content/factor-09-compact-errors.md` | s09 |
| `content/factor-10-small-focused-agents.md` | s10 |
| `content/factor-11-trigger-from-anywhere.md` | s11 |
| `workshops/2025-07-16/walkthrough/12-server.ts` | s11 |
| `content/factor-12-stateless-reducer.md` | s12 |
| `packages/create-12-factor-agent/template/src/agent.ts` | s12 |
| `content/appendix-13-pre-fetch.md` | (not implemented; extension #6) |

---

## Five extension exercises

After reading, try these:

### Exercise 1 — Real Anthropic provider (medium)
**Goal**: make s05 / s06 talk to a real LLM.

Implement an `AnthropicProvider` satisfying `Provider`. Calls `https://api.anthropic.com/v1/messages`. Needs:

- Translate `Thread.SerializeForLLM()` JSON into Anthropic's `messages: [{role, content}]` format
- Parse `tool_use` content blocks back into `NextStep`
- Rate-limit + network-error handling (use s09's self-heal pattern)

Reference: `packages/create-12-factor-agent/template/src/agent.ts`'s BAML `client Anthropic` config.

**Acceptance**: `go test -tags=integration ./...` end-to-end with `ANTHROPIC_API_KEY` set.

### Exercise 2 — SQLite ThreadStore (easy)
**Goal**: persist threads to disk.

Implement `SQLiteStore` satisfying s06's `ThreadStore`. Schema: `threads(id TEXT PRIMARY KEY, events_json TEXT)`. Updates dump the full JSON (no incremental writes; the event count is small).

**Acceptance**: server restart → `GET /thread/{id}` still works.

### Exercise 3 — BAML-style codegen (hard)
**Goal**: derive a JSON schema from your Go `Tool` interface and inject into the prompt.

Write a `go generate` tool that scans `tools.go`, finds every `Tool` implementation's struct fields, emits an OpenAPI-style JSON schema string, and injects it into `promptTemplate` at a `{{ .OutputFormat }}` placeholder.

**Acceptance**: adding a new tool updates the prompt schema automatically; no manual edits.

### Exercise 4 — OpenTelemetry tracing (medium)
**Goal**: emit OTel spans per RunAgent / Provider call / Tool.Execute.

Add `go.opentelemetry.io/otel`. In `loop.go` and `tools.go`, use `ctx.SpanFromContext` + `tracer.Start`. Tag spans with `intent`, `step`, `consecutive_errors`.

**Acceptance**: export traces to Jaeger and see the timeline of each LLM call and tool execution.

### Exercise 5 — Real Slack OAuth trigger (hard)
**Goal**: wire s11's `SlackTrigger` to real Slack.

Need:

- HMAC signature verification (`X-Slack-Signature` + `SLACK_SIGNING_SECRET`)
- handle Slack URL-verification challenge (one-shot GET)
- distinguish `app_mention` events from other messages
- reply to the channel via Slack Web API (`OAuth Bot Token`)

Reference: upstream README's "Trigger from anywhere" section.

**Acceptance**: @-mention the bot in a Slack channel and watch the agent reply.

---

## Where to go next

- **Upstream Discord**: [humanlayer.dev/discord](https://humanlayer.dev/discord) — Dex (author) answers questions there
- **AI Engineer World's Fair talk**: [17-min YouTube version](https://www.youtube.com/watch?v=8kMaTybvDUw) / [deep dive](https://www.youtube.com/watch?v=yxJDyQ8v6P0)
- **shareAI-lab/learn-claude-code** — the pedagogy our repo inherits
- **HumanLayer SDK** — production escalation pipeline (Slack/email routing + SLA tracking) for s07's `RequestApproval`

---

Reading this far means you can build AND criticize 12-factor agents. Happy hacking.
