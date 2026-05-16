package main

import "encoding/json"

type Thread struct {
	Events []Event `json:"events"`
}

func NewThread(seed ...Event) *Thread {
	t := &Thread{Events: make([]Event, 0, 16)}
	t.Events = append(t.Events, seed...)
	return t
}

func (t *Thread) Append(e Event) { t.Events = append(t.Events, e) }

func (t *Thread) LastEvent() (Event, bool) {
	if len(t.Events) == 0 {
		return Event{}, false
	}
	return t.Events[len(t.Events)-1], true
}

func (t *Thread) SerializeForLLM() string {
	out, err := json.MarshalIndent(t.Events, "", "  ")
	if err != nil {
		return `{"error":"thread serialization failed"}`
	}
	return string(out)
}

// AwaitingHuman returns true when the last event is a tool_call whose
// intent is a human-contact intent AND no human_response has been
// appended after it yet.
//
// This is the s07 predicate that turns "looped until done" into "looped
// until done OR awaiting human" without needing a separate state field.
func (t *Thread) AwaitingHuman() bool {
	last, ok := t.LastEvent()
	if !ok {
		return false
	}
	if last.Type != EventTypeToolCall {
		return false
	}
	step, ok := last.Data.(NextStep)
	if !ok {
		return false
	}
	return IsHumanIntent(step.Intent)
}

// IsHumanIntent centralizes the "intents that break the loop" check.
// The closed set lives in tools.go (the tool implementations); this
// function asks the registry-like question without depending on a live
// Registry value.
func IsHumanIntent(intent string) bool {
	switch intent {
	case IntentRequestApproval, IntentRequestMoreInformation:
		return true
	}
	return false
}
