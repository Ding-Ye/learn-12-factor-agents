---
title: "s07 · 用工具调用方式联系人类"
chapter: 7
slug: s07-contact-humans-with-tools
est_read_min: 9
---

# s07 · 用工具调用方式联系人类

> 教什么：人类不是特殊 channel —— 他们就是另一种"工具"。LLM emit `request_approval` 或 `request_more_information` 时，loop **不调用 Execute**，直接退出；HTTP 层把 `response_url` 写进响应，下次 `POST /response` 续传。

---

## Problem / 问题

s06 让 agent 跑通了"启动 / 查询 / 续传"的 HTTP 流程，但 `human_response` 事件还得靠 client 手动 POST 触发——agent 自己并不知道"我应该停下来等人"。真实场景里，LLM 该自己决定"这个 $100k 转账要人审批"。

上游 factor-07 的答案：**让人类成为 tool 的一种**。LLM 看到的 schema 里就有 `RequestApproval`、`AskClarification` 这些 tool；emit 它们时，code 端**不执行**——直接挂起 thread 等 human 回应。

## Solution / 解决方案

3 个决策：

1. **`ErrHumanContact` sentinel error**：tools.go 里定义。RequestApprovalTool.Execute 和 AskClarificationTool.Execute 直接 return 这个 error。RunAgent 用 `errors.Is(err, ErrHumanContact)` 检测后干净退出（不算 error）。
2. **`Thread.AwaitingHuman()` 作判定**：靠 thread 最后一个 event 是不是 human-intent tool_call 推导。和 s05 的 `IsDone` 一样的派生方法。
3. **`ThreadView` 加 `response_url`**：当 awaiting 时填上 `BaseURL + /thread/{id}/response`。Client 不需要知道路由约定 —— 直接 POST 那个 URL。

## How It Works / 工作原理

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

核心 30 行（节选自 [`agents/s07-contact-humans-with-tools/loop.go`](https://github.com/Ding-Ye/learn-12-factor-agents/blob/main/agents/s07-contact-humans-with-tools/loop.go) + tools.go）：

```go
var ErrHumanContact = errors.New("human contact requested")

type RequestApprovalTool struct{}
func (RequestApprovalTool) Intent() string { return IntentRequestApproval }
func (RequestApprovalTool) Execute(_ context.Context, _ json.RawMessage) (any, error) {
    return nil, ErrHumanContact
}

// inside RunAgent ...
result, err := tool.Execute(ctx, next.Data)
if err != nil {
    if errors.Is(err, ErrHumanContact) {
        // append no tool_response — handler will write response_url
        return thread, nil
    }
    return thread, fmt.Errorf("tool %q at step %d: %w", next.Intent, step, err)
}
thread.Append(NewToolResponseEvent(result))
```

**3 个非显然之处**：

1. **`ErrHumanContact` 用 `errors.Is` 检测而不是 type assertion**：未来如果有 wrapper error（`fmt.Errorf("...%w", ErrHumanContact)`），`errors.Is` 还能识别。type assertion 就会 false negative。
2. **human-contact tool 不 append tool_response**：上游 `12-server.ts:31-61` 也是这样 —— `tool_response` 等人类回应到了再有。否则 thread 会出现 `tool_call(approval)` 紧跟 `tool_response(nil)`，LLM 会困惑。
3. **`AwaitingHuman` 在 thread.go，不在 server.go**：predicate 跟 data structure 走。s10 / s12 都会复用同一个 predicate，server 只是其中一个调用者。

## What Changed / 与 s06 的变化

```diff
+ types.go: + RequestApprovalPayload, AskClarificationPayload, 2 new intents
+ thread.go: + AwaitingHuman + IsHumanIntent
+ tools.go: + RequestApprovalTool, AskClarificationTool, ErrHumanContact sentinel
+ loop.go: errors.Is(err, ErrHumanContact) clean exit
+ server.go: ThreadView 加 Awaiting + ResponseURL；view 自动填
- 6 tests (s06)
+ 5 tests (含 AwaitingHuman、loop 早退、HTTP round-trip)
```

语义上的差别：s06 的 loop 退出条件是 `done_for_now` 或 error；s07 加了第三个 `human contact`。三种退出都是 clean (no error)，只是 `AwaitingHuman()` 区分。

## Try It / 动手试一试

```bash
cd agents/s07-contact-humans-with-tools

go test -v -race ./...

go run . :8080 &

curl -s -X POST localhost:8080/thread \
     -H "Content-Type: application/json" \
     -d '{"message":"send 100k to acme"}' | python3 -m json.tool
# 返回包含 "awaiting": true, "response_url": "http://localhost:8080/thread/<id>/response"

# 用 response_url 续传
curl -s -X POST 'http://localhost:8080/thread/<id>/response' \
     -H "Content-Type: application/json" \
     -d '{"message":"approved"}' | python3 -m json.tool
# 续传后 awaiting 消失

kill %1
```

期望输出：第一次 POST 后 thread.events 含 user_input → tool_call(multiply 1000×100) → tool_response(100000) → tool_call(request_approval) → （无 tool_response）。续传后多 human_response + tool_call(done_for_now)。

## Upstream Source Reading / 上游源码阅读

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
            # In notebooks: prompt the user via input(); in server: break
            # and return the thread so the HTTP layer can collect the
            # response asynchronously.
            clarification = clarification_handler(next_step.message)
            thread.events.append({"type": "clarification_response", "data": clarification})
        # ... math tool branches
```

**对照阅读要点**：

- **`ClarificationRequest` 进 union return type**：上游 BAML 在 `DetermineNextStep` 的返回类型 union 里加了 ClarificationRequest 类型 —— LLM 知道这是一个合法的 emit。我们 Go 没有 codegen，靠 prompt 中的 schema 描述告诉 LLM（Phase G 时手写注入）。
- **上游 inline 处理 vs 我们抛 sentinel error**：上游 `if next_step.intent == "request_more_information"` 直接调 `clarification_handler` 收集人类输入；我们 Go 端用 `ErrHumanContact` 让 loop 早退，留给上层 HTTP handler 决定。设计 trade-off：上游同步阻塞、我们异步通过 HTTP。
- **`clarification_response` event** vs **我们的 `human_response`**：上游 event type 是 intent-specific（`clarification_response` / `approval_response`）；我们统一成 `human_response`。简洁，但失了一点 typed structure。
- **`request_approval` 不在上游 BAML**：上游 BAML 只有 `request_more_information`。我们额外加了 `request_approval`（来自 `12-server.ts` 的人类批准流程）。这是教学合理化：approval 比 clarification 更能突出"breaking the loop"。
- **`response_url` 是 humanlayer SDK 概念**：上游 `09-server.ts:23-26` 在返回里写 `response_url`；我们 mirror 这个 convention。

**想读更多**：`content/factor-07-contact-humans-with-tools.md:21-46` + `workshops/2025-07-16/walkthrough/12-server.ts:31-99`（webhook 续传完整流程）。我们 s11 会接 `12-server.ts` 那条线。

---

**下一节预告**：s08 把 loop 里的 "if intent in [...] / else if ... / else if ..." 链显式化成 `ControlFlow(thread, next) Action` 函数，每个分支返回 `ActionLoop` / `ActionBreak` / `ActionEscalate`。新加一个 intent 会变成 compile error，强制开发者明确处理。
