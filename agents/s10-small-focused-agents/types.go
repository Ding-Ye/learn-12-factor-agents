// Package main implements chapter s10: small, focused agents.
//
// Instead of one giant agent with a 50-step context, an Orchestrator
// composes two small agents — CalcAgent (math, 3 turns max) and
// SummaryAgent (writes the final user-facing string). Each sub-agent
// owns its own Thread, keeping contexts small and tests independent.
//
// Upstream reference: content/factor-10-small-focused-agents.md.
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
	IntentSummarize  = "summarize"
)

type MathPayload struct {
	A float64 `json:"a"`
	B float64 `json:"b"`
}

type DoneForNowPayload struct {
	Message string `json:"message"`
}

// SummarizePayload — the orchestrator passes a partial result here for
// the SummaryAgent to turn into a final natural-language message.
type SummarizePayload struct {
	Result float64 `json:"result"`
}
