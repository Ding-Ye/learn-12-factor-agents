---
title: "附录 B · 上游源码导读地图"
slug: appendix-b-upstream-map
est_read_min: 10
---

# 附录 B · 上游源码导读地图

> 12 章 + s_full + 附录 A 读完后，你已经把 12-factor 的核心模式重写了一遍 Go。这份附录告诉你"现在该看上游源码的哪些部分，按什么顺序"。

---

## 推荐阅读顺序

### Step 1 — Manifesto

1. [`README.md`](https://github.com/humanlayer/12-factor-agents/blob/main/README.md) — 上游的"为什么"。重点看前 60 行（agent 框架批判）和 12 factor 列表。
2. [`content/brief-history-of-software.md`](https://github.com/humanlayer/12-factor-agents/blob/main/content/brief-history-of-software.md) — 软件设计史 + agent 在其中的位置。

### Step 2 — Twelve principles in detail

按顺序读 12 个 factor markdown，每个 100-300 行：

| Factor | File | 重点段落 |
|---|---|---|
| 1 | `content/factor-01-natural-language-to-tool-calls.md` | 全文 |
| 2 | `content/factor-02-own-your-prompts.md` | lines 14-91（核心论点） |
| 3 | `content/factor-03-own-your-context-window.md` | lines 69-139（序列化 trade-off） |
| 4 | `content/factor-04-tools-are-structured-outputs.md` | lines 11-50 |
| 5 | `content/factor-05-unify-execution-state.md` | 全文短 |
| 6 | `content/factor-06-launch-pause-resume.md` | 全文 |
| 7 | `content/factor-07-contact-humans-with-tools.md` | lines 21-46 |
| 8 | `content/factor-08-own-your-control-flow.md` | lines 27-68 |
| 9 | `content/factor-09-compact-errors.md` | lines 10-59 |
| 10 | `content/factor-10-small-focused-agents.md` | 全文 |
| 11 | `content/factor-11-trigger-from-anywhere.md` | 全文短 |
| 12 | `content/factor-12-stateless-reducer.md` | 全文（12 行！） |

### Step 3 — TypeScript walkthrough（上游主参考实现）

`workshops/2025-05-17/walkthrough/` 是 BAML + TypeScript 的 12 步实现。先看：

1. [`00-package.json`](https://github.com/humanlayer/12-factor-agents/blob/main/workshops/2025-05-17/walkthrough/00-package.json) — 依赖
2. [`01-agent.baml`](https://github.com/humanlayer/12-factor-agents/blob/main/workshops/2025-05-17/walkthrough/01-agent.baml) + [`01-agent.ts`](https://github.com/humanlayer/12-factor-agents/blob/main/workshops/2025-05-17/walkthrough/01-agent.ts) — 最简 agent（对应我们 s01-s02）
3. [`05-agent.baml`](https://github.com/humanlayer/12-factor-agents/blob/main/workshops/2025-05-17/walkthrough/05-agent.baml) — 加 ClarificationRequest（对应我们 s07）
4. [`09-server.ts`](https://github.com/humanlayer/12-factor-agents/blob/main/workshops/2025-05-17/walkthrough/09-server.ts) + [`09-state.ts`](https://github.com/humanlayer/12-factor-agents/blob/main/workshops/2025-05-17/walkthrough/09-state.ts) — HTTP server + ThreadStore（对应我们 s06）
5. [`12-server.ts`](https://github.com/humanlayer/12-factor-agents/blob/main/workshops/2025-05-17/walkthrough/12-server.ts) — webhook + async loop（对应我们 s11，但上游有 async 模式）

### Step 4 — Python walkthrough

`workshops/2025-07-16/walkthrough/` 是 Python 改编版。质量比 TS 更稳定，notebook 友好。读 `01-agent.py` → `03-agent.py` → `05-agent.py` → `07-agent.py`。

### Step 5 — 生产模板

[`packages/create-12-factor-agent/template/src/`](https://github.com/humanlayer/12-factor-agents/tree/main/packages/create-12-factor-agent/template/src) 是 npm scaffold 出来的"production-ish layout"：

- `agent.ts` 关键文件：lines 89-114 是上游 `agentLoop` 的最完整版本，我们 s12 的 `Reduce` 重构来源
- `state.ts` — ThreadStore 的 production 版（仍 in-memory）
- `cli.ts` / `server.ts` — 各种 entry point

---

## 上游文件 → 我们章节的映射

| 上游文件 | 对应我们的章节 |
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
| `content/appendix-13-pre-fetch.md` | (未做，extension #6) |

---

## 5 个 extension exercise

读完上游后想自己练手，按难度排序：

### Exercise 1 — Real Anthropic provider（中等）
**目标**：让 s05 / s06 跑真 LLM。

实现一个 `AnthropicProvider` 满足 `Provider` 接口，调 `https://api.anthropic.com/v1/messages`。需要：

- 把 `Thread.SerializeForLLM()` 的 JSON 转成 Anthropic API 的 `messages: [{role, content}]` 形式
- 把 LLM 返回的 `tool_use` content block 解出 `NextStep`
- 处理 rate limit / network error（用 s09 的 self-heal 模式）

参考 `packages/create-12-factor-agent/template/src/agent.ts` BAML 的 `client Anthropic` 配置。

**验收**：`go test -tags=integration ./...` 跑通端到端（设 `ANTHROPIC_API_KEY`）。

### Exercise 2 — SQLite ThreadStore（简单）
**目标**：thread 持久化到磁盘。

实现 `SQLiteStore` 满足 s06 的 `ThreadStore` 接口。schema：`threads(id TEXT PRIMARY KEY, events_json TEXT)`。Update 时整个 JSON dump 覆盖（不需要 incremental update — events 数量少）。

**验收**：server 重启后 `GET /thread/{id}` 仍能拿到 thread。

### Exercise 3 — BAML-style codegen（难）
**目标**：从 Go `Tool` 接口 codegen 出 JSON Schema 注入到 prompt。

写一个 `go generate` 工具，扫描 `tools.go` 里所有 `Tool` 实现的 struct 字段，输出 OpenAPI-style JSON Schema 字符串，然后 inject 到 `promptTemplate` 的某个 `{{ .OutputFormat }}` 位置。

**验收**：新加一个 tool 不用改 prompt template，schema 自动更新。

### Exercise 4 — OpenTelemetry tracing（中等）
**目标**：每次 RunAgent / Provider call / Tool.Execute 出一个 OTel span。

加入 `go.opentelemetry.io/otel`，在 `loop.go` / `tools.go` 里 `ctx.SpanFromContext` + `tracer.Start`. Span attributes 带 `intent`, `step`, `consecutive_errors`。

**验收**：跑一个 trace export 到 Jaeger，看见每个 LLM call / tool exec 的 timeline。

### Exercise 5 — Real Slack OAuth trigger（难）
**目标**：把 s11 的 `SlackTrigger` 接上 Slack Bolt。

需要：

- HMAC 签名验证（用 `X-Slack-Signature` + `SLACK_SIGNING_SECRET`）
- handle Slack URL verification challenge（一次性 GET）
- 区分 `app_mention` event 和 message event
- 用 Slack Web API 把 agent 输出 reply 回原 channel（需要 `OAuth Bot Token`）

参考上游 README 的 "Trigger from anywhere" 段。

**验收**：在 Slack workspace @ bot，agent 在 channel 里回 reply。

---

## "去哪里继续学"

- **上游 Discord**：[humanlayer.dev/discord](https://humanlayer.dev/discord) — Dex（作者）经常在里面回问题
- **AI Engineer World's Fair 主题演讲**：[YouTube 链接](https://www.youtube.com/watch?v=8kMaTybvDUw)（17 分钟版） / [深度版](https://www.youtube.com/watch?v=yxJDyQ8v6P0)
- **shareAI-lab/learn-claude-code**：本仓库教学法的灵感来源
- **HumanLayer SDK**：把 s07 的 RequestApproval 接入生产 escalation 流水（Slack/email 路由 + SLA 跟踪）

---

读到这里，你应该既能写 12-factor agent，也能批判性地评价别人的 agent 框架。Happy hacking.
