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

// RunAgent now turns tool errors into thread events so the LLM can
// self-correct. ConsecutiveErrors >= MaxConsecutiveErrors → escalate
// (return with error).
func RunAgent(ctx context.Context, thread *Thread, provider Provider, registry Registry) (*Thread, error) {
	for step := 0; step < MaxSteps; step++ {
		next, err := provider.DetermineNextStep(ctx, thread.SerializeForLLM())
		if err != nil {
			return thread, fmt.Errorf("provider at step %d: %w", step, err)
		}
		thread.Append(NewToolCallEvent(next))

		if next.Intent == IntentDoneForNow {
			return thread, nil
		}

		tool, err := registry.Lookup(next.Intent)
		if err != nil {
			thread.Append(NewErrorEvent(err.Error()))
			if ConsecutiveErrors(thread) >= MaxConsecutiveErrors {
				return thread, fmt.Errorf("escalated after %d consecutive errors", MaxConsecutiveErrors)
			}
			continue
		}

		result, err := SafeExecute(ctx, tool, next.Data)
		if err != nil {
			// Compact error: write the message into the thread instead
			// of returning. The LLM sees it on the next iteration.
			thread.Append(NewErrorEvent(err.Error()))
			if ConsecutiveErrors(thread) >= MaxConsecutiveErrors {
				return thread, fmt.Errorf("escalated after %d consecutive errors", MaxConsecutiveErrors)
			}
			continue
		}

		thread.Append(NewToolResponseEvent(result))
	}
	return thread, fmt.Errorf("MaxSteps=%d reached", MaxSteps)
}
