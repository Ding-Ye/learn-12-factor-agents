# Upstream reading — Factor 12: Stateless reducer

> Source: humanlayer/12-factor-agents @ d20c728.

## Source A: `content/factor-12-stateless-reducer.md` (full — 12 lines)

```markdown
# Factor 12: Make your agent a stateless reducer

Functions all the way down. Your agent is a thread of events. Each
event is processed by a pure function that returns the next thread.
No mutable state outside the thread. No global variables. Just data
flowing through transformations.

Replay becomes trivial — you can re-run any subsequence of events. Fork
becomes trivial — you can branch at any point. Test becomes trivial —
you can assert exact outputs for exact inputs.
```

## Source B: `packages/create-12-factor-agent/template/src/agent.ts` (89-114)

```typescript
// License: Apache 2.0
export async function agentLoop(thread: Thread): Promise<Thread> {
    while (true) {
        const nextStep = await b.DetermineNextStep(thread.serializeForLLM());
        thread.events.push({"type": "tool_call", "data": nextStep});
        if (nextStep.intent === "done_for_now") return thread;
        if (nextStep.intent === "add") {
            const result = nextStep.a + nextStep.b;
            thread.events.push({"type": "tool_response", "data": result});
        }
        // ... more tools
    }
}
```

## Mapping

| Upstream | Our Go | File |
|---|---|---|
| `Promise<Thread>` mutating | `Reduce(Thread, Event) Thread` pure | reducer.go |
| `thread.events.push` | `Thread.Append` (copy-on-write) | thread.go |
| `while (true) { ... await ... }` | `RunAgent` calls Reduce in a loop | loop.go |
| Tool execution in loop | Auto-step inside Reduce | reducer.go |
| `if (intent === "done_for_now")` | `IsDone(thread)` predicate | reducer.go |

## Why our version is truer to the markdown

Upstream's reference implementation (TypeScript) is NOT actually a
pure reducer — it mutates `thread.events`. The markdown describes the
ideal; the impl is convenient. Our Go port closes the gap by:

1. Making `Thread` a value type.
2. `Append` copies its backing array.
3. `Reduce` returns a new Thread; never writes through a pointer.
4. Tests directly assert `a.Equal(b)` after running the same events twice.

## Reading map

- s12 is the closing chapter: it re-frames the whole curriculum as a
  fold over events.
- The integration chapter (`s_full`) demonstrates that all 12 factors
  compose into a single agent run by following one user query end-to-end
  through the codebase.
- Appendix A (Agents are mostly software) is the philosophical close.
- Appendix B (Upstream reading map) tells learners how to read the
  upstream repo top to bottom now that they've finished our port.
