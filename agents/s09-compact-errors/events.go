package main

const (
	EventTypeUserInput    = "user_input"
	EventTypeToolCall     = "tool_call"
	EventTypeToolResponse = "tool_response"
	EventTypeError        = "error"
)

type Event struct {
	Type string      `json:"type"`
	Data interface{} `json:"data"`
}

func NewUserInputEvent(text string) Event {
	return Event{Type: EventTypeUserInput, Data: text}
}

func NewToolCallEvent(step NextStep) Event {
	return Event{Type: EventTypeToolCall, Data: step}
}

func NewToolResponseEvent(result any) Event {
	return Event{Type: EventTypeToolResponse, Data: result}
}

func NewErrorEvent(message string) Event {
	return Event{Type: EventTypeError, Data: message}
}
