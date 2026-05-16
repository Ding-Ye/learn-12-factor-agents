package main

import (
	"context"
	"encoding/json"
	"fmt"
)

// Provider keeps the s01 signature. Implementations choose whether to use
// the rendered prompt or not — but they cannot pretend they didn't see it.
type Provider interface {
	DetermineNextStep(ctx context.Context, serialized string) (NextStep, error)
}

// RecordingProvider is the s02 stub. Unlike s01's EchoProvider it does NOT
// ignore the input — it computes a fingerprint of the rendered prompt and
// echoes that back. Tests use the fingerprint to prove the prompt template
// actually ran (a stale or mutated template produces a different hash).
//
// LastSeen captures the most recent input, so tests can also inspect the
// full rendered text if a hash mismatch occurs.
type RecordingProvider struct {
	LastSeen string
}

func (r *RecordingProvider) DetermineNextStep(_ context.Context, serialized string) (NextStep, error) {
	r.LastSeen = serialized
	payload := DoneForNowPayload{
		Message: fmt.Sprintf("Acknowledged. prompt_hash=%s", PromptHash(serialized)),
	}
	data, err := json.Marshal(payload)
	if err != nil {
		return NextStep{}, err
	}
	return NextStep{Intent: "done_for_now", Data: data}, nil
}
