---
title: "s06 · 启动 / 暂停 / 恢复 HTTP API"
chapter: 6
slug: s06-launch-pause-resume
est_read_min: 10
---

# s06 · 启动 / 暂停 / 恢复 HTTP API

> 教什么：把 s05 的 agent loop 搬到 HTTP 服务里。三个端点：`POST /thread` 启动、`GET /thread/{id}` 看状态、`POST /thread/{id}/response` 续传。

---

## Problem / 问题

s05 让 agent 自己跑通了多步 loop，但只能在 CLI 一次性跑完。真实场景是：用户在 Slack 发起，agent 跑到一半要等审批，半小时后人回来批准了，agent 继续。这需要 (1) 把 thread 存起来；(2) HTTP 启动 / 查询 / 续传三个 endpoint；(3) 多个并发请求互不影响。

上游 factor-06 给的就是这种 thin HTTP 包装。s06 我们用 Go stdlib 实现：`net/http`、`sync.Mutex` 守护的 `map[string]*Thread`、`crypto/rand` 生成 thread id。

## Solution / 解决方案

3 个决策：

1. **`ThreadStore` 是 interface**：In-memory `MemoryStore` 是唯一实现；想换 sqlite/redis 只需新写一个类型。Extension exercise 在 Appendix B。
2. **`RunAgent` 同步跑在 HTTP handler 里**：不开 goroutine、不返回 `{status:"processing"}`。同步让教学场景简单（请求结束 = thread 终态）。生产场景用 async 的 trade-off 留给 s11 commentary 讲。
3. **`POST /response` 复用 `RunAgent`**：appends `human_response` event → 调 `RunAgent` 让 agent 接着跑。s07 会在这个基础上让 agent 真正"在 human 事件后跳过下一次 LLM 调用 / 不跳过"的判断。

## How It Works / 工作原理

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

核心 30 行（节选自 [`agents/s06-launch-pause-resume/server.go`](https://github.com/Ding-Ye/learn-12-factor-agents/blob/main/agents/s06-launch-pause-resume/server.go)）：

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

**3 个非显然之处**：

1. **`http.NewServeMux` + Go 1.22+ 路由模式**：用 `"POST /thread"` 和 `r.PathValue("id")` 就能做出 REST 路由，不用 gorilla/mux。靠的是 Go 1.22 标准库升级。
2. **错误时也 `Store.Update`**：agent 跑挂了我们仍把 thread 存进 store —— 调试时可以 GET 看到最后一个 event。生产代码会上 OpenTelemetry，但教学场景"丢失 = bug"更严重。
3. **`MemoryStore` 用 `sync.Mutex` 不用 `sync.RWMutex`**：写操作占比高（每次 RunAgent 都 Update），RWMutex 的读优化收益不大。教学场景显式 Mutex 比 RWMutex 易读。

## What Changed / 与 s05 的变化

```diff
+ store.go (新建 — ThreadStore + MemoryStore)
+ server.go (新建 — Server + 3 handlers)
+ events.go: + EventTypeHumanResponse + NewHumanResponseEvent
- main.go: CLI demo
+ main.go: HTTP server :8080
- 6 tests (loop_test.go)
+ 6 tests (server_test.go via httptest, 含并发 race 测试)
```

语义上的差别：s05 的"loop"是 CLI 时间线；s06 的"loop"分两段，由 HTTP 请求触发。Thread 第一次跨请求保活。

## Try It / 动手试一试

```bash
cd agents/s06-launch-pause-resume

go test -v -race ./...

# 启动 server
go run . :8080 &

# 启动一个 thread
curl -s -X POST localhost:8080/thread \
     -H "Content-Type: application/json" \
     -d '{"message":"go"}' | python3 -m json.tool

# 用上一步返回的 thread_id 查询
curl -s localhost:8080/thread/<id> | python3 -m json.tool

# 续传（appends human_response，触发 RunAgent 再跑一次）
curl -s -X POST localhost:8080/thread/<id>/response \
     -H "Content-Type: application/json" \
     -d '{"message":"please continue"}' | python3 -m json.tool

# 杀进程
kill %1
```

期望输出：thread_id 是 24 字符 hex，thread.events 包含 user_input + 3 个 tool_call + 2 个 tool_response。续传后多一个 human_response 事件。

## Upstream Source Reading / 上游源码阅读

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

**对照阅读要点**：

- **Express vs net/http**：上游用 express；我们用 stdlib `http.ServeMux`。Go 1.22+ 的 ServeMux 已经能处理 method + path 路由 + path params，不需要 gorilla。
- **`crypto.randomUUID()` vs `crypto/rand`**：上游用 UUID v4（36 字符）；我们用 12 字节 hex（24 字符）。功能等价；UUID 在多机部署时更明确，hex 在 in-memory 单机够用。
- **`async/await` vs Go 同步 + ctx**：上游 express handler 是 async；我们 handler 同步，靠 `r.Context()` 传递 deadline。Go 的同步代码 + goroutine 模型让这里更线性。
- **`thread.events.push` vs `thread.Append`**：上游直接改 array；我们包了个 method。等价。
- **未做的部分**：上游 `12-server.ts:38-58` 演示了"async 模式" —— `POST /thread` 立刻返回 `{status:"processing"}`，agent 在 goroutine 里跑。我们 s06 保持同步，async 模式留给 s11 commentary 讲。

**想读更多**：上游 `workshops/2025-07-16/walkthrough/10-server.ts`（人类批准流程）+ `12-server.ts`（webhook 触发）。我们 s07 / s11 会分别接上这两条线。

---

**下一节预告**：s07 让 agent loop 真正"看见"人类。`RequestApproval` 和 `AskClarification` 进 Tool 接口；当 LLM emit 这两种 intent 之一时，loop **不再调用 tool.Execute**，而是直接返回 thread —— HTTP handler 检测到 `AwaitingHuman` 后把 `response_url` 写进响应，下次 `POST /response` 续传。
