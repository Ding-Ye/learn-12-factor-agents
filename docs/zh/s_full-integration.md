---
title: "s_full · 端到端集成：12 因子贯通走查"
chapter: full
slug: s_full-integration
est_read_min: 14
---

# s_full · 端到端集成：12 因子贯通走查

> 这一章不写新代码 —— 它把 s01..s12 的 12 个因子贯穿到一条真实的 user query 上，让你看见每一行代码对应哪个 factor。

---

## 全栈架构图

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
                            │  Event │ │ Reg│ │             │
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
                                each with its own Thread, tracked via
                                subagent_call / subagent_done events.

                              Own your prompt (s02):
                                text/template renders SYSTEM/USER sections
                                with the serialized Thread.
```

每个 factor 落地：

| Factor | Where it lives in code | Where it shows up at runtime |
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
| 10 Small focused agents | `Orchestrator` + `subagents/` | when task complex |
| 11 Trigger from anywhere | `Trigger` interface + `triggers/` | external entry points |
| 12 Stateless reducer | `Reduce(Thread, Event) Thread` | s12 onward |

---

## 16 步执行轨迹：`"add 5 and 3, then multiply by 2"` + 一次人类批准

为了贯通 12 因子，我们追踪一条复合 query：用户问 add+multiply，agent 算到一半要人类批准是否真的执行第二步。

| 步 | 谁干了什么 | 文件:行 (chapter) | 涉及 factor |
|---|---|---|---|
| 1 | 用户 POST /thread `{message:"..."}` | `s06/server.go:handleCreate` | 6 |
| 2 | Server 创建 Thread，seed `user_input` event | `s03/events.go:NewUserInputEvent` | 3, 5 |
| 3 | Store.Create 返回 thread_id（12 字节 hex） | `s06/store.go:MemoryStore.Create` | 6 |
| 4 | RunAgent 进入 loop，第一次调用 provider | `s05/loop.go:RunAgent` | 5, 8 |
| 5 | Provider 收到 `Thread.SerializeForLLM()` (JSON) | `s03/thread.go:SerializeForLLM` | 3 |
| 6 | LLM/stub 返回 NextStep{Intent:"add", Data:{a:5,b:3}} | `s01/types.go:NextStep` | 1, 4 |
| 7 | ControlFlow 判定 ActionLoop（math intent in registry） | `s08/controlflow.go:ControlFlow` | 8 |
| 8 | Loop append `tool_call` event, run AddTool.Execute → 8 | `s04/tools.go:AddTool` | 4, 5 |
| 9 | Loop append `tool_response` event, continue | `s05/events.go:NewToolResponseEvent` | 5 |
| 10 | 第二次 provider 调用：返回 NextStep{Intent:"request_approval", Data:{question:"approve multiply?"}} | `s07/types.go:RequestApprovalPayload` | 7 |
| 11 | ControlFlow 判定 ActionBreak（human intent） | `s08/controlflow.go` | 7, 8 |
| 12 | RunAgent 返回 thread（仍含 tool_call(request_approval)，无 tool_response） | `s07/loop.go` | 7 |
| 13 | Server 看见 `AwaitingHuman()` 为 true，response 含 `response_url` | `s07/server.go:view` | 7 |
| 14 | 30 分钟后，HumanLayer 发 webhook → POST /triggers/webhook | `s11/triggers/webhook.go` | 11 |
| 15 | WebhookTrigger 解析出 ResumeThreadID + HumanResponse "approved" | `s11/triggers/webhook.go:Trigger` | 11 |
| 16 | Server 取出旧 thread，append `human_response`，再调 RunAgent | `s11/server.go:handleTrigger` | 6, 11 |

后续（第 17-20 步）：Provider 第三次返回 multiply，tool 执行 → tool_response 16，第四次返回 done_for_now，loop 退出。

**用 s12 reducer 的视角看同一过程**：每一步都是 `t = Reduce(t, e)`。整条 16 步轨迹可以 replay：把 16 个 events 重新 fold over 一遍 `Reduce`，得到字节相同的 final Thread。

---

## Deliberate omissions（教学版有意不做的部分）

| 上游有的 | 我们没做 | 原因 |
|---|---|---|
| BAML 代码生成 | 手写 `NextStep{Intent, Data json.RawMessage}` + JSON schema 描述 | Go 没 BAML 等价物；codegen 是分散的话题 |
| 真 Anthropic / OpenAI provider | Stub `ScriptedSequenceProvider` | 每章测试不依赖网络。Phase G 单独引入 |
| 持久化 ThreadStore | in-memory map | sqlite/redis 是基建话题，附录 B 留 exercise |
| HumanLayer SDK 集成 | 自己定义 WebhookTrigger | SDK 学习成本反而压过 trigger 抽象本身 |
| Slack OAuth + HMAC 验签 | 接受任意 POST body | 安全话题独立成 exercise |
| Streaming token response | 一次性 JSON | streaming 涉及 SSE/WS，超出范围 |
| Async agent loop (`/thread` 立刻返 `processing`) | 同步 in-handler | 教学场景同步更线性 |
| Tracing (OpenTelemetry) | 无 | 附录 B exercise |
| BAML's `output_format` injection | 没有 | Provider 端 prompt 自己注入 |
| Pre-fetch / appendix-13 优化 | 无 | 上游也只是 markdown |

---

## 一条 query 跨 12 章的总览

读完这一章你应该能：

1. **回看 s01-s12 的 README**，每一章的 `What Changed vs prev` 段串起来就是这条 16 步 trace 的演化史。
2. **找到任意 factor 的代码位置**：上表"Where it lives in code"列。
3. **添加一个新 tool**（例如 `FetchURL`）：写 struct + 实现 `Tool` 接口 + 加进 `DefaultRegistry()` + 在 `KnownIntents()` 加一行。control-flow 自动走 ActionLoop 分支。
4. **添加一个新 trigger**（例如 Linear webhook）：写 struct + 实现 `Trigger` 接口 + 注册到 server 的 `Triggers` map。
5. **替换 Provider**：从 `ScriptedSequenceProvider` 换成 `OpenAIProvider` 只需一个一行的初始化 swap（前提是 Phase G 引入了 OpenAIProvider）。

附录 A 写"为什么这种设计哲学优于框架包装"。附录 B 给一份完整的上游源码导读路线。
