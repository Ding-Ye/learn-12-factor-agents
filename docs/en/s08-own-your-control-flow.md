---
title: "s08 · Own your control flow"
chapter: 8
slug: s08-own-your-control-flow
est_read_min: 8
---

# s08 · Own your control flow

> What you learn here: the implicit branching from s07 ("if intent ... else if ...") becomes a typed `Action` enum dispatched by a pure `ControlFlow(thread, next, registry) Action` function. Adding an intent without giving it a branch falls through to `ActionEscalate`; `KnownIntents` + an exhaustiveness test guards against the omission.

---

## Problem / The gap

s07's `RunAgent` carried three implicit branches:

- tool.Execute returns `ErrHumanContact` → early exit (no error)
- intent == `done_for_now` → early exit
- everything else → loop

Those branches were scattered inside the loop, mixed with error handling. Adding a new intent (s09's `error` self-heal flow, for example) made it easy to miss a path.

Upstream factor-08's slogan is **own your control flow**: branches should be **explicit + data-driven + enumerable**.

## Solution / Mental model

Three decisions:

1. **`Action` is an enum**: `ActionLoop` / `ActionBreak` / `ActionFinish` / `ActionEscalate`. New branch = add a constant + a case in `ControlFlow`.
2. **`ControlFlow(thread, next, registry) Action` is a pure function**: no I/O, no side effects. The loop dispatches by calling `ControlFlow`, then switches on the returned `Action`.
3. **`KnownIntents()` + exhaustiveness test**: the function enumerates known intents; a test asserts `ControlFlow` returns non-`Invalid` for every one of them. An intent added without a `KnownIntents` entry still trips the runtime path: `ControlFlow` falls through to `default` → `ActionEscalate`.

## How It Works

```
   RunAgent loop:
       ┌────────────────────────────────┐
       │ provider.DetermineNextStep     │
       │   ── append tool_call to thread │
       │   ── action = ControlFlow(...) │
       │                                 │
       │ switch action {                │
       │   case ActionFinish: return     │
       │   case ActionBreak:  return     │
       │   case ActionLoop:              │
       │     ── tool.Execute             │
       │     ── append tool_response     │
       │   case ActionEscalate:          │
       │     ── append error event       │
       │     ── return with error        │
       │ }                               │
       └────────────────────────────────┘
```

Core 30 lines (excerpt from `controlflow.go`):

```go
type Action int
const (
    ActionInvalid Action = iota
    ActionLoop
    ActionBreak
    ActionFinish
    ActionEscalate
)

func ControlFlow(_ *Thread, next NextStep, registry Registry) Action {
    switch next.Intent {
    case IntentDoneForNow:
        return ActionFinish
    case IntentRequestApproval, IntentRequestMoreInformation:
        return ActionBreak
    default:
        if _, ok := registry[next.Intent]; ok {
            return ActionLoop
        }
        return ActionEscalate
    }
}
```

**Three non-obvious bits:**

1. **`ControlFlow` takes no ctx and returns no error** — pure function. Tests can drive every branch in one line (`got := ControlFlow(nil, NextStep{Intent: x}, r)`). All side effects live in `RunAgent`'s switch on `Action`.
2. **`ActionEscalate` doesn't panic** — upstream might fast-fail unknown intents. We escalate because in production the LLM emitting a wrong intent (hallucination) is routine and shouldn't crash the server.
3. **`KnownIntents()` is "documentation as test"** — Go lacks sealed enums, so we hand-list the closed set and let a test assert exhaustiveness. Forget to update the list, the test still passes; forget to update `ControlFlow`, the runtime path catches it via `ActionEscalate`.

## What Changed vs s07

```diff
+ controlflow.go (new — Action enum + ControlFlow + KnownIntents)
- loop.go: scattered if/else early-exit branches
+ loop.go: switch over ControlFlow(...) return value
+ events.go: + EventTypeError + NewErrorEvent (s09 will use heavily)
- 5 tests
+ 7 tests (including exhaustiveness)
```

Semantically: s07's branches were implicit; s08 pins them to `Action` values. Adding an intent now requires explicit annotation of which `Action` it maps to.

## Try It

```bash
cd agents/s08-own-your-control-flow

go test -v ./...

go run .
# Final thread has 6 events:
#   [0] user_input
#   [1] tool_call (add)
#   [2] tool_response
#   [3] tool_call (multiply)
#   [4] tool_response
#   [5] tool_call (request_approval, breaks loop)
```

## Upstream Source Reading

```upstream:workshops/2025-07-16/walkthrough/07-agent.py#L38-L80
# Source: workshops/2025-07-16/walkthrough/07-agent.py lines 38-80
# License: Apache 2.0

def agent_loop(thread: Thread) -> AgentResponse:
    b = get_baml_client()
    while True:
        next_step = b.DetermineNextStep(thread.serialize_for_llm())
        thread.events.append({"type": next_step.intent, "data": next_step})

        if next_step.intent == "done_for_now":
            return next_step
        elif next_step.intent == "request_more_information":
            clarification = clarification_handler(next_step.message)
            thread.events.append({"type": "clarification_response", "data": clarification})
        elif next_step.intent == "fetch_issues":
            issues = await linear_client.issues()
            thread.events.append({"type": "fetch_issues_result", "data": issues})
        elif next_step.intent == "add":
            result = next_step.a + next_step.b
            thread.events.append({"type": "tool_response", "data": result})
        # ... more elifs
```

**Reading notes:**

- **`if/elif` chain vs `Action` enum**: upstream adds an elif per intent; we extract "decision" into one `ControlFlow` function and leave "execution" in the loop. Separation of concerns.
- **Upstream mixes sync and async tools in the same `elif`** — `fetch_issues` is `await`; `add` is sync. We hand the "do I execute?" question to `Action` (`ActionLoop` always executes), which keeps the loop body tidier.
- **Upstream has no `ActionEscalate`** — default behavior on unknown intents is "fall through" → loop re-prompts. We append an error event and escalate, turning "unknown intent" into one explicit failure instead of an endless retry.
- **Exhaustiveness via test** — upstream can't do this in Python; Go can lean on `KnownIntents` for self-check.
- **Upstream uses `await`** — Python asyncio; we use synchronous Go + goroutines.

**Want to read more?** `content/factor-08-own-your-control-flow.md:27-68` argues clearly why control flow doesn't belong to the framework.

---

**Up next, s09:** tool errors stop terminating the agent — they become `error` events on the thread, and the LLM reads them and self-corrects. Three consecutive errors escalate.
