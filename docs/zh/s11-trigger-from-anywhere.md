---
title: "s11 · 任意触发源（Webhook / Slack）"
chapter: 11
slug: s11-trigger-from-anywhere
est_read_min: 8
---

# s11 · 任意触发源（Webhook / Slack）

> 教什么：agent 不该只能从 CLI 启动。`Trigger` 接口接 Slack message、HumanLayer webhook、cron event 等任意外部源；转换成 thread 后下游和 s06 完全一样。

---

## Problem / 问题

s06-s10 都假设 agent 由 `POST /thread {message}` 启动。生产环境里 agent 真正的入口是：

- Slack message（用户 @ bot）
- HumanLayer webhook（人类批准后触发续传）
- cron tick（每天 9 点 trigger 检查邮件 agent）
- Linear webhook（issue 状态变更）
- 邮件转发（IMAP）

如果每加一个入口就写一遍"parse → seed thread → call RunAgent"，会复制粘贴失控。

上游 factor-11 的答案：**把"转换外部 payload"抽成 Trigger 接口**。各 Trigger 实现自己的 parsing，统一产出 `Outcome{FreshUserInput?, ResumeThreadID?, HumanResponse?}`。Server 看 Outcome 决定 spawn 新 thread 还是 resume 旧的。

## Solution / 解决方案

3 个决策：

1. **`Trigger` 接口只两个方法**：`Source() string` 返回名字（用于 trigger event audit），`Trigger(r *http.Request) (Outcome, error)` 解析请求。新加 trigger = 新加一个 struct 实现这两个方法。
2. **`Outcome` 是 union 而不是两个不同的接口**：fresh 用 `FreshUserInput`，resume 用 `ResumeThreadID + HumanResponse`。`IsFresh()` / `IsResume()` 自检。Outcome value semantics 让逻辑测试简单（不需要 mock）。
3. **路由用 `map[string]Trigger`**：`POST /triggers/{name}` lookup → call → process。注册新 trigger 等于改 map 一行。

## How It Works / 工作原理

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

核心 30 行（节选自 [`triggers/types.go`](https://github.com/Ding-Ye/learn-12-factor-agents/blob/main/agents/s11-trigger-from-anywhere/triggers/types.go) + [`server.go`](https://github.com/Ding-Ye/learn-12-factor-agents/blob/main/agents/s11-trigger-from-anywhere/server.go)）：

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

// in server:
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

**3 个非显然之处**：

1. **`Outcome` 用 value semantics 不用 interface**：每个 trigger 返回 typed Outcome。如果让 Outcome 成 interface (`type Outcome interface { Apply(*Server) }`)，能进一步面向对象 —— 但牺牲了"看 server 代码就知道两种分支"的可读性。
2. **`NewTriggerEvent` 进 thread**：trigger source + raw payload 作为审计 breadcrumb。orchestrator/Loop 永远不读它，但调试时知道 thread 是怎么诞生的。
3. **resume 时 trigger 不会创建新 thread**：`outcome.ResumeThreadID` 必须存在；不存在 → 404。这是为了防止"manually-crafted webhook"创建未授权的 thread。

## What Changed / 与 s10 的变化

```diff
+ triggers/ (子包)
+   - types.go: Trigger interface + Outcome
+   - slack.go: SlackTrigger
+   - webhook.go: WebhookTrigger (HumanLayer-style)
+ events.go: + EventTypeTrigger + NewTriggerEvent
- server.go: POST /thread (s06-style)
+ server.go: POST /triggers/{name} (路由按 Trigger map)
- 6 tests
+ 5 tests (含 trigger 双路径 + 错误形态)
```

语义上的差别：s10 是"orchestrator 拆 sub-agent"；s11 是"trigger 接外部"。两者都是为了 isolation —— sub-agent 隔离 context，trigger 隔离入口。

## Try It / 动手试一试

```bash
cd agents/s11-trigger-from-anywhere

go test -v ./...

go run . :8080 &

# Slack 风格 trigger
curl -s -X POST localhost:8080/triggers/slack \
     -H "Content-Type: application/json" \
     -d '{"event":{"type":"message","text":"add 2 and 3","channel":"C1"}}' | python3 -m json.tool

# 用上一步返回的 thread_id 测试 webhook resume
curl -s -X POST localhost:8080/triggers/webhook \
     -H "Content-Type: application/json" \
     -d '{"event":{"spec":{"state":{"thread_id":"<id>"}},"status":{"response":"ok"}}}' | python3 -m json.tool

kill %1
```

期望输出：第一次 trigger 返回 thread_id + thread.events（含 trigger event + user_input event + 完整 loop）；第二次 webhook 在已有 thread 上 append `human_response`。

## Upstream Source Reading / 上游源码阅读

```upstream:workshops/2025-07-16/walkthrough/12-server.ts#L31-L99
// Source: workshops/2025-07-16/walkthrough/12-server.ts lines 31-99
// License: Apache 2.0

app.post('/thread', async (req: Request, res: Response) => {
    const body = req.body as V1Beta2EmailEventReceived | { message: string };
    let thread: Thread;
    
    if ('event' in body) {
        // Email-triggered: extract sender/subject/body into a user_input
        const event = body.event;
        thread = new Thread([{ type: 'email_received', data: event }]);
    } else {
        // CLI-style: plain { message }
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

**对照阅读要点**：

- **上游 `if ('event' in body)` 嵌进 `/thread` handler**：判定类型靠 duck typing。我们 Go 端把判定提到 `Trigger.Trigger` 接口外，server 只看 Outcome。Go 没有 duck typing 反推促成更清晰的边界。
- **上游 `/webhook` 是独立 endpoint**：每种 trigger 一个 endpoint。我们用 `/triggers/{name}` 一个 endpoint 路由所有 trigger。前者更"显式"，后者更"可扩展"。哪种好取决于 trigger 数量（5 个以下用前者更易读；30 个用后者必要）。
- **未做的部分**：上游 `V1Beta2HumanContactCompleted` 是 HumanLayer SDK 类型 —— 它会做 HMAC 验签、retry-after 头、idempotency key 等。我们 minimal 实现都没做。Appendix B 列了 #5 extension exercise 把这些补上。
- **`response.event.spec.state?.thread_id`**：上游用 `?.` 链式访问；我们 Go 端用 struct 嵌套 + decode 一步到位。
- **缺的部分**：cron / IMAP 等长连接 / 拉取式 trigger 上游也没实现 —— 留给读者。

**想读更多**：上游 `content/factor-11-trigger-from-anywhere.md` 一段就讲清了 idea；`12-server.ts` 整个文件是参考实现。

---

**下一节预告**：s12 把 RunAgent 从"维护 mutable Thread 指针"重构成纯 `Reduce(Thread, Event) Thread` —— 同样的输入永远出同样的输出，replay + fork 测试一行就能写。
