---
title: "s12 · Stateless reducer (replay + fork)"
chapter: 12
slug: s12-stateless-reducer
est_read_min: 9
---

# s12 · Stateless reducer (replay + fork)

> What you learn here: refactor the whole agent loop into a pure `Reduce(Thread, Event) Thread`. No side effects, no globals. Replay is one line; fork is one line.

---

## Problem / The gap

s05-s11 always used a `*Thread` pointer. Append is O(1), but:

- "run the same thread twice and get the same result" is hard to test because the thread is mutated.
- Forking a thread into two branches requires hand-written deep copy.
- Steps can't be isolated — any line could mutate the thread.

Upstream factor-12's claim: **an agent is a reducer**. `f(state, event) = next_state`. Like Redux: a fold over events.

## Solution / Mental model

Three decisions:

1. **`Reduce(Thread, Event) Thread` takes values, returns values** — no pointer, no error return. Internal failures (unmarshal errors) become `error` events on the returned thread. Errors are data.
2. **`Thread.Append` always copies a new slice** — `make([]Event, len+1)` + `copy`. O(n), but forks never alias. Production code might use a persistent vector for O(log n) appends.
3. **`Reduce` auto-steps tool execution after a `tool_call`** — one `Reduce` call represents one complete "decide + execute" cycle. Replay tests don't need to simulate "two adjacent events" — each Reduce settles fully.

## How It Works

```
   Thread{}  ──► Reduce(t, user_input)
              ──► Reduce(t, tool_call(add 5,3))
                    │
                    └── auto-step: compute 5+3=8, append tool_response(8)
              ──► Reduce(t, tool_call(multiply 8,2))
                    │
                    └── auto-step: compute 8*2=16, append tool_response(16)
              ──► Reduce(t, tool_call(done_for_now))
                    │
                    └── (no auto-step; loop terminates)
              ──► IsDone(t) == true
```

Core 30 lines (excerpt from `reducer.go`):

```go
func Reduce(t Thread, e Event) Thread {
    t = t.Append(e)

    if e.Type != EventTypeToolCall {
        return t
    }
    var step NextStep
    if err := json.Unmarshal(e.Data, &step); err != nil {
        return t.Append(NewEvent("error", fmt.Sprintf("decode tool_call: %v", err)))
    }
    switch step.Intent {
    case IntentAdd:
        var p MathPayload
        _ = json.Unmarshal(step.Data, &p)
        return t.Append(NewEvent(EventTypeToolResponse, p.A+p.B))
    case IntentMultiply:
        var p MathPayload
        _ = json.Unmarshal(step.Data, &p)
        return t.Append(NewEvent(EventTypeToolResponse, p.A*p.B))
    case IntentDoneForNow:
        return t  // terminal
    default:
        return t.Append(NewEvent("error", fmt.Sprintf("unknown intent %q", step.Intent)))
    }
}

func ReduceMany(t Thread, events []Event) Thread {
    for _, e := range events { t = Reduce(t, e) }
    return t
}
```

**Three non-obvious bits:**

1. **Thread is a value type** — copying it is cheap (a slice header is 24 bytes). `func (t Thread) Append` returns a new Thread; the old one stays unchanged. That's immutable-by-convention.
2. **`Reduce` takes no ctx and no provider** — a pure function. Tool execution is hard-wired into the reducer for teaching simplicity. Production code would keep the IO outside, but the principle (pure reducer + impure boundary) holds.
3. **`Equal` uses JSON marshal instead of `reflect.DeepEqual`** — `json.RawMessage` is a `[]byte` and `DeepEqual` has nil-vs-empty-slice quirks. JSON marshalling abstracts "are these byte-identical?"

## What Changed vs s11

```diff
+ reducer.go: Reduce + ReduceMany + IsDone (chapter's star)
+ thread.go: Thread becomes value-type; Append returns a new Thread; + Equal
- events.go: Data changes from interface{} to json.RawMessage
- loop.go: RunAgent shrinks to a shell around Reduce
- 5 tests
+ 6 tests (replay + fork + no-mutation + auto-step)
```

Semantically: before s12, the loop body **mutated** the thread; now it **computes** the next thread. Small change in spelling, big change in testability / replayability / forkability.

## Try It

```bash
cd agents/s12-stateless-reducer

go test -v ./...
# 6 PASS: replay, fork, no-mutation, auto-step, end-to-end

go run .
# Final thread (6 events):
#   [0] user_input: "add 5 and 3, then multiply by 2"
#   [1] tool_call: {"intent":"add","data":{"a":5,"b":3}}
#   [2] tool_response: 8
#   [3] tool_call: {"intent":"multiply","data":{"a":8,"b":2}}
#   [4] tool_response: 16
#   [5] tool_call: {"intent":"done_for_now","data":{"message":"Result is 16."}}
```

Note how the replay test is just three lines:

```go
a := ReduceMany(Thread{}, events)
b := ReduceMany(Thread{}, events)
if !a.Equal(b) { t.Fatal("replay failed") }
```

That brevity is what makes a pure reducer powerful — the property comes for free.

## Upstream Source Reading

```upstream:packages/create-12-factor-agent/template/src/agent.ts#L89-L114
// Source: packages/create-12-factor-agent/template/src/agent.ts lines 89-114
// License: Apache 2.0

export async function agentLoop(thread: Thread): Promise<Thread> {
    while (true) {
        const nextStep = await b.DetermineNextStep(thread.serializeForLLM());

        thread.events.push({
            "type": "tool_call",
            "data": nextStep
        });

        if (nextStep.intent === "done_for_now") {
            return thread;
        }

        if (nextStep.intent === "add") {
            const result = nextStep.a + nextStep.b;
            thread.events.push({"type": "tool_response", "data": result});
        }
        // ... more tools
    }
}
```

```upstream:content/factor-12-stateless-reducer.md#L1-L12
# Source: content/factor-12-stateless-reducer.md lines 1-12
# License: CC BY-SA 4.0

# Factor 12: Make your agent a stateless reducer

Functions all the way down. Your agent is a thread of events. Each
event is processed by a pure function that returns the next thread.
No mutable state outside the thread. No global variables. Just data
flowing through transformations.

Replay becomes trivial — you can re-run any subsequence of events. Fork
becomes trivial — you can branch at any point. Test becomes trivial —
you can assert exact outputs for exact inputs.
```

**Reading notes:**

- **Upstream's `agentLoop` still mutates** — `thread.events.push(...)` directly modifies the array. The markdown describes a reducer as the aspiration; the TypeScript reference implementation is not actually pure. Our Go port realizes the aspiration.
- **`thread.events.push` is O(1) amortized in TypeScript** — but immutability is broken. Our Go port copies (O(n)) but stays immutable. A persistent vector library would close the gap.
- **Upstream's "functions all the way down" slogan** — our `Reduce + ReduceMany + IsDone` are all pure. The slogan holds.
- **Upstream's `await` is inside the loop** — IO in the loop body. We move IO out (RunAgent's caller); `Reduce` itself never touches IO. A teaching bonus.
- **What upstream wishes for but doesn't fully ship**: replay + fork. Our Go reducer tests actually run them. That's the chapter's payoff.

**Want to read more?** `content/factor-12-stateless-reducer.md` (12 lines) plus `packages/create-12-factor-agent/template/src/agent.ts:89-114`. Read with React/Redux reducer docs alongside — the "agent is an LLM-driven reducer" mental model is the same shape.

---

## End of curriculum

You finished s01 → s12. See the integration chapter (`s_full`), Appendix A (Agents are mostly software), and Appendix B (Upstream reading map) under `docs/`.
