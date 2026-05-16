// Package main implements chapter s07: contact humans with tools.
//
// We add two new tool intents — request_approval and
// request_more_information — that DON'T execute on the agent side.
// Instead, the loop appends the tool_call event and returns. The HTTP
// handler detects `Thread.AwaitingHuman()`, sets `response_url` on the
// reply, and the human's response comes back through `POST /response`.
//
// Upstream reference: workshops/2025-07-16/walkthrough/05-agent.baml:30-33
// + workshops/2025-07-16/walkthrough/05-agent.py:24-36.
package main

import "encoding/json"

type NextStep struct {
	Intent string          `json:"intent"`
	Data   json.RawMessage `json:"data"`
}

const (
	IntentAdd                    = "add"
	IntentMultiply               = "multiply"
	IntentDoneForNow             = "done_for_now"
	IntentRequestApproval        = "request_approval"
	IntentRequestMoreInformation = "request_more_information"
)

type MathPayload struct {
	A float64 `json:"a"`
	B float64 `json:"b"`
}

type DoneForNowPayload struct {
	Message string `json:"message"`
}

// RequestApprovalPayload — payload for the human-approval tool.
type RequestApprovalPayload struct {
	Question string `json:"question"`
	Stakes   string `json:"stakes,omitempty"`
}

// AskClarificationPayload — payload for the clarification tool.
type AskClarificationPayload struct {
	Message string `json:"message"`
}
