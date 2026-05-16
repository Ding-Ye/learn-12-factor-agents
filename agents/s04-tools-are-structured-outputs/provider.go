package main

import (
	"context"
	"encoding/json"
	"strings"
)

type Provider interface {
	DetermineNextStep(ctx context.Context, serialized string) (NextStep, error)
}

// ScriptedProvider returns canned NextStep values keyed on user input
// substrings. This is still deterministic (no LLM call) but now we can
// drive arithmetic dispatch from tests.
//
// In s05 we'll add a multi-step scripted provider; for now, one shot.
type ScriptedProvider struct{}

func (ScriptedProvider) DetermineNextStep(_ context.Context, prompt string) (NextStep, error) {
	prompt = strings.ToLower(prompt)
	switch {
	case strings.Contains(prompt, "add ") || strings.Contains(prompt, "+"):
		return mathStep(IntentAdd, 2, 3)
	case strings.Contains(prompt, "subtract") || strings.Contains(prompt, "minus") || strings.Contains(prompt, "-"):
		return mathStep(IntentSubtract, 5, 2)
	case strings.Contains(prompt, "multiply") || strings.Contains(prompt, "times") || strings.Contains(prompt, "*"):
		return mathStep(IntentMultiply, 4, 6)
	case strings.Contains(prompt, "divide") || strings.Contains(prompt, "/"):
		return mathStep(IntentDivide, 10, 2)
	default:
		data, _ := json.Marshal(DoneForNowPayload{Message: "Nothing to do."})
		return NextStep{Intent: IntentDoneForNow, Data: data}, nil
	}
}

func mathStep(intent string, a, b float64) (NextStep, error) {
	data, err := json.Marshal(MathPayload{A: a, B: b})
	if err != nil {
		return NextStep{}, err
	}
	return NextStep{Intent: intent, Data: data}, nil
}
