package main

import (
	"context"
	"encoding/json"
	"fmt"
)

const MaxSteps = 16

type Provider interface {
	DetermineNextStep(ctx context.Context, serialized string) (Event, error)
}

// ScriptedSequenceProvider returns canned events in order. In s12 the
// provider's job is to emit a tool_call event; Reduce takes care of the
// tool_response.
type ScriptedSequenceProvider struct {
	Events []Event
	calls  int
}

func (s *ScriptedSequenceProvider) DetermineNextStep(_ context.Context, _ string) (Event, error) {
	if s.calls >= len(s.Events) {
		return NewEvent(EventTypeToolCall, doneStep("exhausted")), nil
	}
	e := s.Events[s.calls]
	s.calls++
	return e, nil
}

func mathToolCall(intent string, a, b float64) Event {
	step := NextStep{Intent: intent}
	step.Data, _ = json.Marshal(MathPayload{A: a, B: b})
	return NewEvent(EventTypeToolCall, step)
}

func doneStep(message string) NextStep {
	step := NextStep{Intent: IntentDoneForNow}
	step.Data, _ = json.Marshal(DoneForNowPayload{Message: message})
	return step
}

func doneToolCall(message string) Event { return NewEvent(EventTypeToolCall, doneStep(message)) }

// RunAgent is now a thin shell around Reduce: pull next event, apply.
// All the "what to do with this event" logic lives in Reduce.
func RunAgent(ctx context.Context, t Thread, provider Provider) (Thread, error) {
	for step := 0; step < MaxSteps; step++ {
		e, err := provider.DetermineNextStep(ctx, t.SerializeForLLM())
		if err != nil {
			return t, fmt.Errorf("provider at step %d: %w", step, err)
		}
		t = Reduce(t, e)
		if IsDone(t) {
			return t, nil
		}
	}
	return t, fmt.Errorf("MaxSteps=%d reached", MaxSteps)
}
