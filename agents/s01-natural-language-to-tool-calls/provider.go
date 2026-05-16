package main

import (
	"context"
	"encoding/json"
)

// Provider is the one-method interface every chapter implements. In s01 the
// only implementation is EchoProvider, which returns a hardcoded NextStep so
// we can exercise the wire format without any LLM call. From s02 onward the
// provider receives a rendered prompt; from s03 onward the input is a
// serialized Thread.
type Provider interface {
	DetermineNextStep(ctx context.Context, serialized string) (NextStep, error)
}

// EchoProvider is the stub LLM. Given any input it always returns the same
// done_for_now NextStep. This is enough to prove the round-trip works and
// to give the rest of the chapter a foothold for tests.
//
// The hardcoded reply mirrors the dossier's quickstart output:
//
//	{ intent: 'done_for_now', message: 'Hello! How can I assist you today?' }
type EchoProvider struct{}

// DefaultMessage is exported for tests so they can assert against the same
// canonical string the EchoProvider returns. Keeping it as a package-level
// constant makes the assertion location obvious in test failure output.
const DefaultMessage = "Hello! How can I assist you today?"

// DetermineNextStep ignores the input and returns a fixed done_for_now step.
// The error return is part of the interface so that real providers introduced
// in later chapters (Phase G) can surface transport / parse failures without
// changing the interface shape.
func (EchoProvider) DetermineNextStep(_ context.Context, _ string) (NextStep, error) {
	data, err := json.Marshal(DoneForNowPayload{Message: DefaultMessage})
	if err != nil {
		// Marshalling a struct with only string fields is infallible in
		// practice, but we surface the error rather than panicking to keep
		// the interface honest.
		return NextStep{}, err
	}
	return NextStep{
		Intent: "done_for_now",
		Data:   data,
	}, nil
}
