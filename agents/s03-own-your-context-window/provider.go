package main

import (
	"context"
	"encoding/json"
	"strings"
)

type Provider interface {
	DetermineNextStep(ctx context.Context, serialized string) (NextStep, error)
}

// EchoThreadProvider acknowledges the thread it received. It scans the
// thread for "user_input" events and parrots back a short summary, then
// returns done_for_now. This is enough to prove the Thread → serialize →
// prompt → provider round-trip in tests.
type EchoThreadProvider struct{}

func (EchoThreadProvider) DetermineNextStep(_ context.Context, rendered string) (NextStep, error) {
	// Count "user_input" occurrences in the rendered prompt so the
	// returned message reflects what the provider actually saw. The
	// counting is intentionally crude — Phase G's real provider doesn't
	// need it.
	count := strings.Count(rendered, `"type": "user_input"`)
	message := "Thread received."
	if count > 0 {
		message = "Thread received with " + pluralize(count, "user_input event") + "."
	}
	data, err := json.Marshal(DoneForNowPayload{Message: message})
	if err != nil {
		return NextStep{}, err
	}
	return NextStep{Intent: "done_for_now", Data: data}, nil
}

func pluralize(n int, singular string) string {
	if n == 1 {
		return "1 " + singular
	}
	return formatInt(n) + " " + singular + "s"
}

// formatInt avoids strconv to keep imports minimal and obvious.
func formatInt(n int) string {
	if n == 0 {
		return "0"
	}
	digits := []byte{}
	for n > 0 {
		digits = append([]byte{byte('0' + n%10)}, digits...)
		n /= 10
	}
	return string(digits)
}
