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

func mathStep(a, b float64) NextStep {
	data, _ := json.Marshal(MathPayload{A: a, B: b})
	return NextStep{Intent: IntentAdd, Data: data}
}

func doneStep(message string) NextStep {
	data, _ := json.Marshal(DoneForNowPayload{Message: message})
	return NextStep{Intent: IntentDoneForNow, Data: data}
}

// RunAgent — minimal s05-shaped loop. Triggers route here.
func RunAgent(ctx context.Context, thread *Thread, provider Provider) (*Thread, error) {
	for step := 0; step < MaxSteps; step++ {
		next, err := provider.DetermineNextStep(ctx, thread.SerializeForLLM())
		if err != nil {
			return thread, err
		}
		thread.Append(NewToolCallEvent(next))
		if next.Intent == IntentDoneForNow {
			return thread, nil
		}
		switch next.Intent {
		case IntentAdd:
			var p MathPayload
			if err := json.Unmarshal(next.Data, &p); err != nil {
				return thread, err
			}
			thread.Append(NewToolResponseEvent(p.A + p.B))
		default:
			return thread, fmt.Errorf("unknown intent %q", next.Intent)
		}
	}
	return thread, fmt.Errorf("MaxSteps reached")
}
