package main

import "encoding/json"

type Thread struct {
	Events []Event
}

func NewThread(seed ...Event) *Thread {
	t := &Thread{Events: make([]Event, 0, 16)} // bigger preallocation for multi-step runs
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

// IsDone returns true when the most recent event is a tool_call for
// done_for_now. This is the canonical "loop should exit" predicate —
// it derives execution state purely from thread events, no side state.
func IsDone(t *Thread) bool {
	last, ok := t.LastEvent()
	if !ok {
		return false
	}
	if last.Type != EventTypeToolCall {
		return false
	}
	if step, ok := last.Data.(NextStep); ok {
		return step.Intent == IntentDoneForNow
	}
	return false
}

// LastToolCall returns the most recent tool_call event's NextStep. Useful
// for tests asserting which intent the agent just fired.
func LastToolCall(t *Thread) (NextStep, bool) {
	for i := len(t.Events) - 1; i >= 0; i-- {
		if t.Events[i].Type == EventTypeToolCall {
			if step, ok := t.Events[i].Data.(NextStep); ok {
				return step, true
			}
		}
	}
	return NextStep{}, false
}
