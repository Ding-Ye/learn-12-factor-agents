// Package main implements chapter s11: trigger from anywhere.
//
// External event sources (Slack webhooks, HumanLayer events, cron, etc.)
// produce threads through a uniform `Trigger` interface. Once the
// trigger emits a thread, downstream looks identical to s06's flow.
//
// Upstream reference: content/factor-11-trigger-from-anywhere.md +
// workshops/2025-07-16/walkthrough/12-server.ts.
package main

import "encoding/json"

type NextStep struct {
	Intent string          `json:"intent"`
	Data   json.RawMessage `json:"data"`
}

const (
	IntentAdd        = "add"
	IntentDoneForNow = "done_for_now"
)

type MathPayload struct {
	A float64 `json:"a"`
	B float64 `json:"b"`
}

type DoneForNowPayload struct {
	Message string `json:"message"`
}
