// Package main implements chapter s03 of learn-12-factor-agents.
//
// Goal: own the context window. The "input" to the provider is no longer a
// raw string — it's a serialized Thread of Event values that the agent
// builds up turn by turn. The same Thread will carry tool calls, tool
// responses, errors and human inputs in later chapters.
//
// Upstream reference:
//   - content/factor-03-own-your-context-window.md (the why)
//   - workshops/2025-07-16/walkthrough/01-agent.py lines 7-19 (Event + Thread)
package main

// Event types we use in this chapter. As later chapters introduce more
// kinds of events, the closed set grows — but every event's wire shape
// stays `{type, data}` so serialization stays stable.
const (
	EventTypeUserInput    = "user_input"
	EventTypeToolCall     = "tool_call"
	EventTypeToolResponse = "tool_response"
)

// Event is one entry in the Thread. The Type field is the discriminator.
// Data is `any` for ergonomics, but every constructor we provide produces
// strongly-typed values so writers never need to deal in maps.
//
// After JSON round-trip, Data widens to map[string]any — that's a known
// limitation we document and revisit in s12.
type Event struct {
	Type string      `json:"type"`
	Data interface{} `json:"data"`
}

// NewUserInputEvent builds an event from a raw user message.
// Construction helpers keep call sites readable: `NewUserInputEvent("hi")`
// beats `Event{Type:"user_input", Data:"hi"}` at the read site.
func NewUserInputEvent(text string) Event {
	return Event{Type: EventTypeUserInput, Data: text}
}

// NewToolCallEvent and NewToolResponseEvent live here so all the closed
// set of event constructors are visible in one file.
func NewToolCallEvent(payload interface{}) Event {
	return Event{Type: EventTypeToolCall, Data: payload}
}

func NewToolResponseEvent(result interface{}) Event {
	return Event{Type: EventTypeToolResponse, Data: result}
}
