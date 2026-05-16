// Package main implements chapter s02 of learn-12-factor-agents.
//
// Goal: stop pretending the prompt is the framework's business. We render
// the prompt **explicitly** via Go's text/template so every token reaching
// the LLM is something the developer wrote on purpose.
//
// Upstream reference: workshops/2025-07-16/walkthrough/01-agent.baml lines
// 11-27 (the BAML prompt block) and content/factor-02-own-your-prompts.md
// lines 14-91 (the rationale).
package main

import "encoding/json"

// NextStep is the same tagged-union value as s01 — the wire format is the
// one thing that stays constant across chapters.
type NextStep struct {
	Intent string          `json:"intent"`
	Data   json.RawMessage `json:"data"`
}

// DoneForNowPayload mirrors the BAML DoneForNow class. Unchanged from s01.
type DoneForNowPayload struct {
	Message string `json:"message"`
}
