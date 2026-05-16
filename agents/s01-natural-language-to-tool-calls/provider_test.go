package main

import (
	"context"
	"encoding/json"
	"testing"
)

// TestEchoProvider_DefaultIntent verifies that EchoProvider always returns
// intent="done_for_now" regardless of input. This is the canonical wire
// behaviour the rest of the curriculum relies on.
func TestEchoProvider_DefaultIntent(t *testing.T) {
	provider := EchoProvider{}
	cases := []string{
		"",
		"hello",
		"add 2 and 3",
		"can you add 3 and 4, then add 6 to that result",
	}
	for _, in := range cases {
		step, err := provider.DetermineNextStep(context.Background(), in)
		if err != nil {
			t.Fatalf("input %q: unexpected error: %v", in, err)
		}
		if step.Intent != "done_for_now" {
			t.Errorf("input %q: intent = %q, want %q", in, step.Intent, "done_for_now")
		}
	}
}

// TestEchoProvider_PayloadMessage decodes the payload and checks that it
// carries the canonical greeting string. Future chapters will replace this
// with a real LLM message, so pinning the exact text here documents the
// stub's contract.
func TestEchoProvider_PayloadMessage(t *testing.T) {
	step, err := EchoProvider{}.DetermineNextStep(context.Background(), "hello")
	if err != nil {
		t.Fatalf("DetermineNextStep: %v", err)
	}
	var payload DoneForNowPayload
	if err := json.Unmarshal(step.Data, &payload); err != nil {
		t.Fatalf("unmarshal payload: %v", err)
	}
	if payload.Message != DefaultMessage {
		t.Errorf("payload.Message = %q, want %q", payload.Message, DefaultMessage)
	}
}

// TestNextStep_JSONRoundTrip ensures NextStep survives marshal/unmarshal
// without losing the discriminator or the payload bytes. Later chapters
// transmit NextStep across HTTP boundaries (s06), so byte-stable round-trip
// matters now.
func TestNextStep_JSONRoundTrip(t *testing.T) {
	original, err := EchoProvider{}.DetermineNextStep(context.Background(), "")
	if err != nil {
		t.Fatalf("DetermineNextStep: %v", err)
	}

	encoded, err := json.Marshal(original)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	var decoded NextStep
	if err := json.Unmarshal(encoded, &decoded); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	if decoded.Intent != original.Intent {
		t.Errorf("intent mismatch: got %q, want %q", decoded.Intent, original.Intent)
	}
	if string(decoded.Data) != string(original.Data) {
		t.Errorf("data mismatch: got %s, want %s", string(decoded.Data), string(original.Data))
	}
}

// TestRenderNextStep_DoneForNow exercises the CLI's renderNextStep helper
// against the canonical step. Keeping the helper testable means we never
// need to spawn a subprocess to verify the user-visible output.
func TestRenderNextStep_DoneForNow(t *testing.T) {
	step, err := EchoProvider{}.DetermineNextStep(context.Background(), "")
	if err != nil {
		t.Fatalf("DetermineNextStep: %v", err)
	}
	out, err := renderNextStep(step)
	if err != nil {
		t.Fatalf("renderNextStep: %v", err)
	}
	want := `intent=done_for_now message="Hello! How can I assist you today?"`
	if out != want {
		t.Errorf("renderNextStep output:\n got: %s\nwant: %s", out, want)
	}
}

// TestRenderNextStep_UnknownIntent shows the fallback path so a learner who
// edits NextStep.Intent during exploration still gets a sensible printout.
func TestRenderNextStep_UnknownIntent(t *testing.T) {
	step := NextStep{
		Intent: "future_tool",
		Data:   json.RawMessage(`{"a":1}`),
	}
	out, err := renderNextStep(step)
	if err != nil {
		t.Fatalf("renderNextStep: %v", err)
	}
	want := `intent=future_tool data={"a":1}`
	if out != want {
		t.Errorf("renderNextStep output:\n got: %s\nwant: %s", out, want)
	}
}
