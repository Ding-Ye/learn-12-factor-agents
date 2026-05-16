package main

import (
	"encoding/json"
	"fmt"
)

// Reduce is the pure function at the heart of s12. Given a Thread and
// an Event, it returns the next Thread. No side effects, no global
// state, no IO. The "agent loop" is just `for !done { t = Reduce(t, e); e = next }`.
//
// Even tool execution lives behind a thin shim function so the math is
// part of the reducer's domain knowledge, not an arbitrary call.
func Reduce(t Thread, e Event) Thread {
	t = t.Append(e)

	// Auto-step: if this event is a tool_call for a math intent, the
	// reducer also appends the tool_response. This is what makes
	// Reduce-driven loops feel synchronous: each user event "settles"
	// into a stable state before the next one.
	if e.Type != EventTypeToolCall {
		return t
	}

	var step NextStep
	if err := json.Unmarshal(e.Data, &step); err != nil {
		return t.Append(NewEvent("error", fmt.Sprintf("decode tool_call: %v", err)))
	}

	switch step.Intent {
	case IntentAdd:
		var p MathPayload
		_ = json.Unmarshal(step.Data, &p)
		return t.Append(NewEvent(EventTypeToolResponse, p.A+p.B))
	case IntentMultiply:
		var p MathPayload
		_ = json.Unmarshal(step.Data, &p)
		return t.Append(NewEvent(EventTypeToolResponse, p.A*p.B))
	case IntentDoneForNow:
		// done_for_now has no tool_response; the loop will see this
		// terminal step and stop.
		return t
	default:
		return t.Append(NewEvent("error", fmt.Sprintf("unknown intent %q", step.Intent)))
	}
}

// ReduceMany applies Reduce in sequence. Useful for replay tests:
//
//	t1 := ReduceMany(Thread{}, events)
//	t2 := ReduceMany(Thread{}, events)
//	t1.Equal(t2) == true   // because Reduce is pure
func ReduceMany(t Thread, events []Event) Thread {
	for _, e := range events {
		t = Reduce(t, e)
	}
	return t
}

// IsDone returns true when the most recent event is a done_for_now
// tool_call (the canonical terminal state).
func IsDone(t Thread) bool {
	last, ok := t.LastEvent()
	if !ok || last.Type != EventTypeToolCall {
		return false
	}
	var step NextStep
	if err := json.Unmarshal(last.Data, &step); err != nil {
		return false
	}
	return step.Intent == IntentDoneForNow
}
