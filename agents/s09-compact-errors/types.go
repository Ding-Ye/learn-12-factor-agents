// Package main implements chapter s09: compact errors into context.
//
// In s08 a tool error terminated the agent. Now we append an `error`
// event to the thread instead — the LLM sees it next iteration and
// (typically) self-corrects. We cap consecutive errors at 3 to prevent
// retry storms.
//
// Upstream reference: content/factor-09-compact-errors.md +
// workshops/2025-07-16/walkthrough/03-agent.py:21-35.
package main

import "encoding/json"

type NextStep struct {
	Intent string          `json:"intent"`
	Data   json.RawMessage `json:"data"`
}

const (
	IntentAdd                    = "add"
	IntentMultiply               = "multiply"
	IntentDivide                 = "divide"
	IntentDoneForNow             = "done_for_now"
	IntentRequestApproval        = "request_approval"
	IntentRequestMoreInformation = "request_more_information"
)

const MaxConsecutiveErrors = 3

type MathPayload struct {
	A float64 `json:"a"`
	B float64 `json:"b"`
}

type DoneForNowPayload struct {
	Message string `json:"message"`
}
