package main

const (
	EventTypeUserInput     = "user_input"
	EventTypeToolCall      = "tool_call"
	EventTypeToolResponse  = "tool_response"
	EventTypeHumanResponse = "human_response"
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

// NewHumanResponseEvent is the s06 addition — POST /thread/{id}/response
// appends one of these so the agent can resume with the human's reply.
// We don't actually break the loop on human intents yet (that's s07);
// the event is here so resume tests have something to assert against.
func NewHumanResponseEvent(text string) Event {
	return Event{Type: EventTypeHumanResponse, Data: text}
}
