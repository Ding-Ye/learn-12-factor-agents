# Upstream reading — Factor 7: Contact humans with tools

> Source: humanlayer/12-factor-agents @ d20c728.

## Source A: `workshops/2025-07-16/walkthrough/05-agent.baml` (29-37)

```baml
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

## Source B: `workshops/2025-07-16/walkthrough/05-agent.py` (24-40)

```python
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
```

## Mapping

| Upstream | Our Go | File |
|---|---|---|
| `ClarificationRequest` BAML class | `AskClarificationTool` + `AskClarificationPayload` | tools.go / types.go |
| (not in upstream BAML) | `RequestApprovalTool` (s07 adds this) | tools.go |
| Inline `if` branch in `agent_loop` | `ErrHumanContact` sentinel | tools.go |
| Synchronous `clarification_handler(...)` | HTTP-asynchronous `POST /response` | server.go |
| `clarification_response` event | `human_response` (unified) | events.go |

## Reading map

- s07 → s08: the loop's branches become typed `Action` values.
- s07 → s11: triggers (Slack / webhook) become alternative entry points
  that produce `user_input` events instead of going through `/thread`.
- HumanLayer SDK in production: `RequestApproval`'s `Stakes` field
  drives routing (e.g., `low` auto-approves after 5 min, `high` always
  requires a human).
