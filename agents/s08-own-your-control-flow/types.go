// Package main implements chapter s08: own your control flow.
//
// s07's RunAgent had implicit branching: "tool returned ErrHumanContact?
// break". s08 makes the dispatch table a typed Action enum and a named
// ControlFlow function. Adding a new intent without giving it a branch
// turns into a compile error.
//
// Upstream reference: content/factor-08-own-your-control-flow.md +
// workshops/2025-07-16/walkthrough/07-agent.py:38-100.
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

type RequestApprovalPayload struct {
	Question string `json:"question"`
	Stakes   string `json:"stakes,omitempty"`
}

type AskClarificationPayload struct {
	Message string `json:"message"`
}
