---
title: "附录 A · 智能体即软件（Agents are mostly software）"
slug: appendix-a-agents-are-software
est_read_min: 12
---

# 附录 A · 智能体即软件

> 上游 12-factor agents 不只是 12 条工程模式 —— 它本质上是对"agent 框架"的批判。本附录把那条主线拎出来。

---

## 一句话：框架反转的反转

agent 框架（langchain / crewai 等）说"给我个 prompt + 一袋工具，剩下我来"；上游 12-factor 说"剩下你自己来，框架最多给你一个 LLM client"。

整本课程的 12 个 factor 都是"自己来"的具体落地：

| 框架做的事 | 12-factor 让你做的事 | 对应章节 |
|---|---|---|
| 替你写 prompt | 你自己用 `text/template` 写 | s02 |
| 替你管 context window | 你自己 append 到 `Thread.Events` | s03 |
| 替你 register tools | 你自己 implement `Tool` interface | s04 |
| 替你跑 agent loop | 你自己写 `for {}` 调 RunAgent | s05 |
| 替你管 sessions | 你自己开 `ThreadStore` map | s06 |
| 替你 handle humans | 你自己定义 `RequestApproval` tool | s07 |
| 替你调度 | 你自己写 `Action` enum + switch | s8 |
| 替你 retry | 你自己 append `error` event + 数 streak | s9 |
| 替你 plan multi-step | 你自己写 orchestrator | s10 |
| 替你接 Slack | 你自己写 Trigger | s11 |
| 替你管 state | 你自己 fold reducer | s12 |

为什么"自己来"反而是正确的：

---

## 1. 框架的失败模式

读上游 README（第 40-50 行）的论点：

> Most of the products out there billing themselves as "AI Agents" are not all that agentic. A lot of them are mostly deterministic code, with LLM steps sprinkled in at just the right points to make the experience truly magical.

换句话说：**真正能产品化的 agent 大都是普通软件 + 少量 LLM 调用**。框架想替你打包"agent loop"，把 LLM 调用包成 magic，结果：

1. **prompt 透明度丢失**：框架内部组装的 system prompt 你看不见。模型表现差时，你 debug 不到根因。
2. **control flow 被绑死**：框架决定"打 LLM 一次 → tool 调一次 → 循环"。但生产场景里，你常需要"打 LLM 一次 → 做 5 个 deterministic API call → 再打一次 LLM"。框架不让你插入。
3. **state model 是黑盒**：框架的 memory / session / conversation 抽象一般包了一层 ORM。出问题时你要先理解框架的 state，才能 debug 你的业务 state。
4. **failure mode 隐藏**：tool 失败时框架内部 retry 几次？怎么 retry？要不要告诉用户？这些决策原本是产品决策，被框架替你做了。

上游 factor-08（own your control flow）专门讲这点：

> Most agent frameworks treat the agent loop as the magic at the center. But in production, the agent loop is exactly where you need the most control — for retries, for cancellation, for cost limits, for human approvals.

---

## 2. 微 agent 模式（s10 的精神）

12-factor factor-10 说 agent 应该 3-20 步。超过这个范围，两类失败模式必然出现：

- **context bloat**：每步往 prompt 加 tokens；20 步后 prompt 自身的"信噪比"低到 LLM 开始幻觉
- **mixed concerns**：50 步的 agent 在做很多不同的事，prompt 互相干扰

我们 s10 通过 orchestrator 把任务拆给 CalcAgent + SummaryAgent。每个子 agent **自己**只有 3-5 步，**自己**的 thread 不互相干扰。Orchestrator 是 deterministic Go 代码 —— 不是 LLM，因为"什么时候算、什么时候总结、按什么顺序"是 product knowledge，写代码比 prompt 工程便宜得多。

类比 Unix philosophy：

> Make each program do one thing well. To do a new job, build afresh rather than complicate old programs by adding new "features".

12-factor 的"小而专一" = Unix 的"do one thing well"。

---

## 3. 与 langchain / crewai 的对照

举两个例子：

### langchain `AgentExecutor`

```python
agent = create_react_agent(llm, tools, prompt)
executor = AgentExecutor(agent=agent, tools=tools, verbose=True)
result = executor.invoke({"input": "what is 2 + 2?"})
```

- prompt 在 `create_react_agent` 内部组装。你能看见模板，但模板里有 hardcoded 的 reasoning trace 格式
- loop 在 `AgentExecutor.invoke` 里。你能传 callback 但不能改 loop 的退出条件
- 失败处理：`AgentExecutor` 内部 retry 3 次，每次返 raw exception。你不能让 LLM 在 retry 之间看到错误自我修复

vs. 我们 s09 的 `RunAgent`：

```go
for step := 0; step < MaxSteps; step++ {
    next, err := provider.DetermineNextStep(ctx, thread.SerializeForLLM())
    if err != nil { return thread, ... }
    thread.Append(NewToolCallEvent(next))
    // ControlFlow / SafeExecute / append error / continue ...
}
```

每一行都看得见。改 retry 行为 = 改一行 if。

### crewai `Crew`

```python
crew = Crew(agents=[researcher, writer], tasks=[research_task, write_task])
result = crew.kickoff()
```

- "researcher → writer" 顺序在 `tasks` 列表里固定
- 中间数据怎么传？crewai 内部有 `output_pydantic` 之类的抽象，但跨 agent 的 data shape 不是你掌控的
- 想让 writer 在错时 fallback 到另一个 agent？你得读 crewai 源码找 hooks

vs. 我们 s10 的 `Orchestrate`：

```go
calcOut, err := subagents.CalcAgent(plan)
if err != nil { return "", err }  // 想 fallback？这里加 if
thread.Append(NewSubAgentDoneEvent("CalcAgent", calcOut))
// ... pass to SummaryAgent
```

控制流就是普通 Go 代码。

---

## 4. "production-grade" 在 12-factor 语境里意味着什么

读完 12 章你会发现，"production-grade agent" 不是用了什么牛 LLM，而是：

1. **可观测**：每一步都在 `Thread.Events` 里，可以 dump 出来分析
2. **可重放**：s12 的 reducer 保证 replay 字节一致 —— 复现 bug 不用重新跑 LLM
3. **deterministic error handling**：s9 让错误进 thread，retry 次数可控，escalation 路径明确
4. **human-in-loop typed**：s7 让人类批准走和 API call 同样的代码路径（同一个 Tool 接口）
5. **trigger 解耦**：s11 让"哪里来的请求"和"agent 做什么"完全分开。Slack / cron / webhook 通过同一个 ThreadStore

这五条做下来，agent 是普通服务。Pager 半夜响起来时，debug 流程和 debug 任何其他 microservice 一样：看 thread log、找异常事件、写测试复现。

---

## 5. 反对意见与回应

**"自己写所有东西不慢吗？"**
慢，慢 1-2 周。但生产化阶段框架的 magic 也要花同样多时间去理解 + 反向工程 + 替换。前期省的不一定后期省。

**"那 LangGraph 呢？它也讲 state machine"**
LangGraph 比 LangChain 更靠近 12-factor —— 它把 control flow 暴露给你。但它仍带一套"runtime" + "state schema" + checkpoint store。如果你的业务和 LangGraph 假设的 state shape 不吻合，你要么扭曲业务、要么 fork LangGraph。12-factor 没有 runtime，没有 schema，只有 Go 代码。

**"那我永远不该用框架？"**
prototype 阶段用框架挺好。验证想法、试 prompt、看 LLM 反应。**产品化**阶段把它替换掉 —— 这正是上游 factor-01 (Natural Language to Tool Calls) 想说的：framework 的 NL→tool 是 demo 级，production 级需要 typed structure + own dispatch。

---

## 6. 一句话总结

> Agents are mostly software. Don't let the framework hide the software from you.

读完 12 章 + 这篇附录后，你 agent 工程的工具箱里多了一种 stance：**框架质疑者**。下次看 langchain / crewai 的 README，问自己：

- prompt 在哪？我能 diff 它吗？
- thread/memory 数据结构是啥？我能 serialize 它吗？
- loop 在哪？我能加 if 吗？
- 失败时框架默认 retry 几次？我能改吗？

如果四个答案都是 "no/不行"，那就该手写。

---

附录 B 把上游源码读法做成 reading map。
