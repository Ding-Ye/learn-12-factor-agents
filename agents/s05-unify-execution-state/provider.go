package main

import (
	"context"
	"encoding/json"
	"fmt"
)

type Provider interface {
	DetermineNextStep(ctx context.Context, serialized string) (NextStep, error)
}

// ScriptedSequenceProvider returns a sequence of canned NextStep values
// in order — first call returns Steps[0], second returns Steps[1], etc.
// When the sequence is exhausted it returns done_for_now.
//
// This lets tests drive the loop through multiple turns without any LLM.
// It is the s05 evolution of s04's single-shot ScriptedProvider.
type ScriptedSequenceProvider struct {
	Steps []NextStep
	calls int
}

func (s *ScriptedSequenceProvider) DetermineNextStep(_ context.Context, _ string) (NextStep, error) {
	if s.calls >= len(s.Steps) {
		data, _ := json.Marshal(DoneForNowPayload{Message: fmt.Sprintf("Done after %d turns.", s.calls)})
		return NextStep{Intent: IntentDoneForNow, Data: data}, nil
	}
	step := s.Steps[s.calls]
	s.calls++
	return step, nil
}

// Calls reports how many turns the provider has served — useful for
// tests asserting "the loop made exactly 3 LLM calls".
func (s *ScriptedSequenceProvider) Calls() int { return s.calls }

// Helper constructors keep test setup readable.
func mathStep(intent string, a, b float64) NextStep {
	data, _ := json.Marshal(MathPayload{A: a, B: b})
	return NextStep{Intent: intent, Data: data}
}

func doneStep(message string) NextStep {
	data, _ := json.Marshal(DoneForNowPayload{Message: message})
	return NextStep{Intent: IntentDoneForNow, Data: data}
}
