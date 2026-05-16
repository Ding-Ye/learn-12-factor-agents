// Package main implements chapter s04: tools are structured outputs.
//
// Upstream factor-04 says the LLM should never emit free text we have to
// parse — it should emit typed objects. We model that with a `Tool`
// interface + concrete tool structs (`AddTool`, `SubtractTool`,
// `MultiplyTool`, `DivideTool`, `DoneForNow`) and dispatch by Intent.
//
// Upstream reference:
//   - content/factor-04-tools-are-structured-outputs.md (concepts)
//   - workshops/2025-07-16/walkthrough/05-agent.baml (the four math tools)
package main

import (
	"encoding/json"
)

type NextStep struct {
	Intent string          `json:"intent"`
	Data   json.RawMessage `json:"data"`
}

// Closed set of intents the s04 stub provider can emit. Future chapters
// add more (request_more_information in s07, error in s09, etc.).
const (
	IntentAdd        = "add"
	IntentSubtract   = "subtract"
	IntentMultiply   = "multiply"
	IntentDivide     = "divide"
	IntentDoneForNow = "done_for_now"
)

// MathPayload is the shared shape for arithmetic tools. Upstream BAML has
// four separate classes (AddTool, SubtractTool, MultiplyTool, DivideTool)
// — they all carry the same `a, b` fields, so we factor the payload.
// The discriminator lives on NextStep.Intent, not on the payload.
type MathPayload struct {
	A float64 `json:"a"`
	B float64 `json:"b"`
}

type DoneForNowPayload struct {
	Message string `json:"message"`
}
