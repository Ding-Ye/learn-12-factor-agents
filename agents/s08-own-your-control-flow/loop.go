package main

import (
	"context"
	"encoding/json"
	"fmt"
)

const MaxSteps = 16

type Provider interface {
	DetermineNextStep(ctx context.Context, serialized string) (NextStep, error)
}

type ScriptedSequenceProvider struct {
	Steps []NextStep
	calls int
}

func (s *ScriptedSequenceProvider) DetermineNextStep(_ context.Context, _ string) (NextStep, error) {
	if s.calls >= len(s.Steps) {
		data, _ := json.Marshal(DoneForNowPayload{Message: "Sequence exhausted."})
		return NextStep{Intent: IntentDoneForNow, Data: data}, nil
	}
	step := s.Steps[s.calls]
	s.calls++
	return step, nil
}

func mathStep(intent string, a, b float64) NextStep {
	data, _ := json.Marshal(MathPayload{A: a, B: b})
	return NextStep{Intent: intent, Data: data}
}

func doneStep(message string) NextStep {
	data, _ := json.Marshal(DoneForNowPayload{Message: message})
	return NextStep{Intent: IntentDoneForNow, Data: data}
}

func approvalStep(q, s string) NextStep {
	data, _ := json.Marshal(RequestApprovalPayload{Question: q, Stakes: s})
	return NextStep{Intent: IntentRequestApproval, Data: data}
}

// RunAgent now defers the routing decision to ControlFlow; the body is
// a clean for-switch over actions.
func RunAgent(ctx context.Context, thread *Thread, provider Provider, registry Registry) (*Thread, error) {
	for step := 0; step < MaxSteps; step++ {
		next, err := provider.DetermineNextStep(ctx, thread.SerializeForLLM())
		if err != nil {
			return thread, fmt.Errorf("provider at step %d: %w", step, err)
		}
		thread.Append(NewToolCallEvent(next))

		switch ControlFlow(thread, next, registry) {
		case ActionFinish:
			return thread, nil

		case ActionBreak:
			return thread, nil

		case ActionLoop:
			tool := registry[next.Intent] // ControlFlow already proved this exists
			result, err := tool.Execute(ctx, next.Data)
			if err != nil {
				thread.Append(NewErrorEvent(err.Error()))
				return thread, fmt.Errorf("tool %q at step %d: %w", next.Intent, step, err)
			}
			thread.Append(NewToolResponseEvent(result))

		case ActionEscalate:
			thread.Append(NewErrorEvent(fmt.Sprintf("no route for intent %q", next.Intent)))
			return thread, fmt.Errorf("escalated: no route for intent %q", next.Intent)

		default:
			// Defensive: a future Action value missing a case here is
			// almost certainly a bug. Surface it instead of silently
			// looping.
			return thread, fmt.Errorf("control flow returned unhandled action at step %d", step)
		}
	}
	return thread, fmt.Errorf("MaxSteps=%d reached", MaxSteps)
}
