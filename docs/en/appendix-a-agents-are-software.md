---
title: "Appendix A · Agents are mostly software"
slug: appendix-a-agents-are-software
est_read_min: 12
---

# Appendix A · Agents are mostly software

> The 12 factors aren't just engineering patterns — they're a critique of agent frameworks. This appendix pulls the through-line into the foreground.

---

## One-line thesis: the inversion of the inversion

Agent frameworks (langchain / crewai etc.) say "hand me a prompt + a bag of tools; I'll do the rest." Upstream 12-factor says "you do the rest; the framework gives you at most an LLM client."

Every factor is a place where "you do it" lands:

| Frameworks do | 12-factor lets you do | Chapter |
|---|---|---|
| Author the prompt | Render it with `text/template` | s02 |
| Manage the context window | Append to `Thread.Events` | s03 |
| Register tools | Implement the `Tool` interface | s04 |
| Run the agent loop | Write the `for {}` calling RunAgent | s05 |
| Manage sessions | Run a `ThreadStore` map | s06 |
| Handle humans | Define `RequestApproval` tool | s07 |
| Dispatch | Author the `Action` switch | s08 |
| Retry | Append `error` events, count streak | s09 |
| Plan multi-step | Write the orchestrator | s10 |
| Wire Slack/webhooks | Implement the Trigger | s11 |
| Manage state | Fold a reducer | s12 |

Why "do it yourself" is the right answer:

---

## 1. Framework failure modes

From the upstream README (lines 40–50):

> Most of the products out there billing themselves as "AI Agents" are not all that agentic. A lot of them are mostly deterministic code, with LLM steps sprinkled in at just the right points to make the experience truly magical.

In other words: **production agents are usually ordinary software + a few LLM calls**. Frameworks try to package the "agent loop" as magic, which costs you:

1. **Prompt transparency**: the framework's internal system prompt is opaque. When the model misbehaves, you can't diff to the root cause.
2. **Locked control flow**: the framework chose "LLM → tool → loop." Production often wants "LLM → 5 deterministic API calls → LLM again." Frameworks resist inserting that.
3. **Black-box state**: framework memory/session/conversation abstractions usually wrap an ORM. To debug your business state you have to understand the framework's state first.
4. **Hidden failure modes**: how many retries? With backoff? Surface to user? Those are product decisions the framework made for you.

From factor-08 (own your control flow):

> Most agent frameworks treat the agent loop as the magic at the center. But in production, the agent loop is exactly where you need the most control — for retries, for cancellation, for cost limits, for human approvals.

---

## 2. The micro-agent pattern (s10's spirit)

Factor-10 caps agents at 3–20 steps. Beyond that:

- **Context bloat**: every step adds tokens; past 20 steps the signal-to-noise ratio drops to hallucination territory
- **Mixed concerns**: a 50-step agent juggles many sub-tasks; prompts interfere

s10 splits work across `CalcAgent` + `SummaryAgent`. Each sub-agent runs 3–5 steps in **its own** thread; contexts don't bleed. The orchestrator is deterministic Go code — not an LLM — because "when to compute, when to summarize, in what order" is product knowledge cheaper as code than as prompt engineering.

Compare with the Unix philosophy:

> Make each program do one thing well. To do a new job, build afresh rather than complicate old programs by adding new "features".

12-factor's "small focused" = Unix's "do one thing well."

---

## 3. Contrast with langchain / crewai

### langchain `AgentExecutor`

```python
agent = create_react_agent(llm, tools, prompt)
executor = AgentExecutor(agent=agent, tools=tools, verbose=True)
result = executor.invoke({"input": "what is 2 + 2?"})
```

- prompt assembled inside `create_react_agent`. You see the template but it embeds a hardcoded reasoning-trace format
- loop inside `AgentExecutor.invoke`. Callbacks allowed; rewriting the exit condition is not
- failure handling: `AgentExecutor` retries 3 times internally, returns raw exceptions. The LLM can't see prior errors and self-correct between retries

vs. our s09 `RunAgent`:

```go
for step := 0; step < MaxSteps; step++ {
    next, err := provider.DetermineNextStep(ctx, thread.SerializeForLLM())
    if err != nil { return thread, ... }
    thread.Append(NewToolCallEvent(next))
    // ControlFlow / SafeExecute / append error / continue ...
}
```

Every line visible. Changing retry behavior = editing one `if`.

### crewai `Crew`

```python
crew = Crew(agents=[researcher, writer], tasks=[research_task, write_task])
result = crew.kickoff()
```

- "researcher → writer" sequence baked into the `tasks` list
- data flow between agents goes through crewai's `output_pydantic`-style abstractions you don't control
- want writer to fall back to a different agent on failure? Read crewai source for hooks

vs. our s10 `Orchestrate`:

```go
calcOut, err := subagents.CalcAgent(plan)
if err != nil { return "", err }  // want a fallback? Add an if here.
thread.Append(NewSubAgentDoneEvent("CalcAgent", calcOut))
// ... pass to SummaryAgent
```

Control flow is just Go.

---

## 4. What "production-grade" means in 12-factor terms

After 12 chapters, "production-grade agent" doesn't mean using a fancy LLM. It means:

1. **Observable**: every step is in `Thread.Events`. Dump and analyze.
2. **Replayable**: s12's reducer guarantees byte-identical replays — reproducing a bug doesn't re-spend LLM tokens.
3. **Deterministic error handling**: s09 puts errors into the thread; retry counts are bounded; escalation paths are explicit.
4. **Typed human-in-loop**: s07 routes human approvals through the same `Tool` interface as API calls.
5. **Trigger decoupled**: s11 makes "where the request came from" orthogonal to "what the agent does." Slack / cron / webhook share one ThreadStore.

When the pager fires at 2am, debugging an agent looks like debugging any microservice: read the thread log, find the abnormal event, write a test to reproduce.

---

## 5. Objections + responses

**"Writing all that yourself is slow."**
Yes, 1-2 weeks slower up front. But during productionization the framework's magic costs the same time in understanding + reverse-engineering + replacing. Savings up front may not last.

**"What about LangGraph? It exposes a state machine."**
LangGraph is closer to 12-factor than LangChain — it exposes control flow. But it still ships a runtime + state schema + checkpoint store. If your business's state shape doesn't match LangGraph's, you either bend your business or fork LangGraph. 12-factor has no runtime, no schema, just Go.

**"So I should never use a framework?"**
At prototype stage frameworks are great. Validate ideas, try prompts, watch the LLM react. **Productionization** is when you replace it — which is exactly factor-01's point: framework "NL→tool" is demo grade; production grade needs typed structures + own dispatch.

---

## 6. One-line takeaway

> Agents are mostly software. Don't let the framework hide the software from you.

After 12 chapters + this appendix, your agent-engineering toolbox has a new stance: **framework skeptic**. The next time you read a langchain / crewai README, ask:

- Where's the prompt? Can I diff it?
- What's the thread/memory data structure? Can I serialize it?
- Where's the loop? Can I add an `if`?
- How many retries by default? Can I change it?

If the answers are no/no/no/no — hand-write it.

---

Appendix B turns the upstream reading into a guided map.
