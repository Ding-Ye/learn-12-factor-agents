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

// ConsecutiveErrors counts trailing `error` events (paired with a
// preceding tool_call). It walks back from the end of the thread,
// counts how many tool_call → error pairs sit at the tail, and stops
// at the first non-error event.
//
// This is the s09 derived state: "how shaky are we right now?" The
// counter resets the moment we see a successful tool_response or any
// non-error event.
func ConsecutiveErrors(t *Thread) int {
	count := 0
	// Walk back in pairs (tool_call, error). A non-error event breaks
	// the streak.
	for i := len(t.Events) - 1; i >= 0; i-- {
		switch t.Events[i].Type {
		case EventTypeError:
			count++
		case EventTypeToolCall:
			// tool_call paired with error continues the streak; if it
			// would NOT have an error after it (e.g., last event is
			// tool_call), we stop here.
			continue
		default:
			return count
		}
	}
	return count
}
