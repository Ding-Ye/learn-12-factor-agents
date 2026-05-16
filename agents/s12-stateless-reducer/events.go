package main

import "encoding/json"

const (
	EventTypeUserInput    = "user_input"
	EventTypeToolCall     = "tool_call"
	EventTypeToolResponse = "tool_response"
)

type Event struct {
	Type string `json:"type"`
	// Data uses json.RawMessage in s12 (not interface{}) because
	// `Reduce` needs to compare events for identity via JSON bytes.
	// interface{} unmarshals to map[string]any which has non-stable
	// iteration in test diffs.
	Data json.RawMessage `json:"data"`
}

func NewEvent(eventType string, data any) Event {
	b, _ := json.Marshal(data)
	return Event{Type: eventType, Data: b}
}

// Reading note: this struct is intentionally the only mutation surface.
// All "writes" in s12 go through `Reduce` which RETURNS a new Thread.
