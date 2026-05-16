package main

const (
	EventTypeUserInput    = "user_input"
	EventTypeToolCall     = "tool_call"
	EventTypeToolResponse = "tool_response"
)

type Event struct {
	Type string      `json:"type"`
	Data interface{} `json:"data"`
}

func NewUserInputEvent(text string) Event {
	return Event{Type: EventTypeUserInput, Data: text}
}

// NewToolCallEvent now wraps the NextStep so the thread carries the full
// intent + payload — important when the LLM (or scripted provider) needs
// to see what it did in earlier turns.
func NewToolCallEvent(step NextStep) Event {
	return Event{Type: EventTypeToolCall, Data: step}
}

func NewToolResponseEvent(result any) Event {
	return Event{Type: EventTypeToolResponse, Data: result}
}
