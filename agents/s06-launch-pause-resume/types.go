// Package main implements chapter s06: launch / pause / resume HTTP API.
//
// We host the s05 agent loop behind a tiny HTTP service so threads can be
// started, inspected, and (in s07) resumed across process restarts (the
// in-memory store keeps them only as long as the process lives — that's
// the explicit teaching gap).
//
// Upstream reference: content/factor-06-launch-pause-resume.md +
// workshops/2025-07-16/walkthrough/09-server.ts + 09-state.ts.
package main

import "encoding/json"

type NextStep struct {
	Intent string          `json:"intent"`
	Data   json.RawMessage `json:"data"`
}

const (
	IntentAdd        = "add"
	IntentSubtract   = "subtract"
	IntentMultiply   = "multiply"
	IntentDivide     = "divide"
	IntentDoneForNow = "done_for_now"
)

type MathPayload struct {
	A float64 `json:"a"`
	B float64 `json:"b"`
}

type DoneForNowPayload struct {
	Message string `json:"message"`
}
