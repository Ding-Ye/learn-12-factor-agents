package main

const (
	EventTypeUserInput    = "user_input"
	EventTypeToolCall     = "tool_call"
	EventTypeToolResponse = "tool_response"
	EventTypeSubAgentCall = "subagent_call"
	EventTypeSubAgentDone = "subagent_done"
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

// NewSubAgentCallEvent and NewSubAgentDoneEvent record orchestrator-level
// activity so a reader of the orchestrator's thread can see "we
// delegated to CalcAgent at step 1, got back 16 at step 2."
func NewSubAgentCallEvent(name string, input any) Event {
	return Event{Type: EventTypeSubAgentCall, Data: map[string]any{"agent": name, "input": input}}
}

func NewSubAgentDoneEvent(name string, output any) Event {
	return Event{Type: EventTypeSubAgentDone, Data: map[string]any{"agent": name, "output": output}}
}
