// Package main implements chapter s01 of learn-12-factor-agents.
//
// Goal of this chapter: nail the wire format between the LLM and the agent
// without yet talking to a real LLM. We define the minimal types so the rest
// of the curriculum has a stable foundation to build on:
//
//   - NextStep: the tagged-union value an LLM returns; the Intent field is
//     the discriminator and Data holds the payload as raw JSON.
//   - Provider: the single-method interface every later chapter implements.
//
// Upstream reference: workshops/2025-07-16/walkthrough/01-agent.baml lines
// 1-27 (the DoneForNow class + DetermineNextStep function) and
// workshops/2025-07-16/walkthrough/01-agent.py lines 23-26 (the call site).
package main

import "encoding/json"

// NextStep is the value the Provider returns. Intent is a free-form string
// chosen from a closed set known to the agent ("done_for_now", "add", etc.).
// Data is the per-intent payload encoded as raw JSON so that future chapters
// can swap shapes without changing this type.
//
// Mirrors the BAML pattern `class XTool { intent "x"; ... }` where the
// discriminator literal lives on the value itself.
type NextStep struct {
	Intent string          `json:"intent"`
	Data   json.RawMessage `json:"data"`
}

// DoneForNowPayload is the canonical shape that ships with intent
// "done_for_now". It is the only payload type defined this chapter; later
// chapters add AddPayload / SubtractPayload / etc.
//
// Upstream: workshops/2025-07-16/walkthrough/01-agent.baml lines 6-9:
//
//	class DoneForNow {
//	  intent "done_for_now"
//	  message string
//	}
type DoneForNowPayload struct {
	Message string `json:"message"`
}
