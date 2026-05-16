// Package main implements chapter s12: stateless reducer.
//
// The whole loop is re-expressed as `Reduce(Thread, Event) Thread`.
// No global state, no mutable pointers — same input always produces
// the same output. Replay and fork tests become one-liners.
//
// Upstream reference: content/factor-12-stateless-reducer.md +
// packages/create-12-factor-agent/template/src/agent.ts:89-114.
package main

import "encoding/json"

type NextStep struct {
	Intent string          `json:"intent"`
	Data   json.RawMessage `json:"data"`
}

const (
	IntentAdd        = "add"
	IntentMultiply   = "multiply"
	IntentDoneForNow = "done_for_now"
)

type MathPayload struct {
	A float64 `json:"a"`
	B float64 `json:"b"`
}

type DoneForNowPayload struct {
	Message string `json:"message"`
}
