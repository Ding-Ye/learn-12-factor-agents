// Package main implements chapter s05: unify execution state with business state.
//
// All chapters from now on accumulate state in `Thread.Events`. We never
// introduce a parallel "agent state" map. To answer "is the agent still
// running?", "what tool just ran?", "how many errors in a row?" we query
// the thread.
//
// Upstream reference: content/factor-05-unify-execution-state.md +
// workshops/2025-07-16/walkthrough/03-agent.py:14-37.
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
