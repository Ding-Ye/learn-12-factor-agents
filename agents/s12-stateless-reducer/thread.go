package main

import (
	"bytes"
	"encoding/json"
)

// Thread is a value type in s12 (not a pointer). Reduce takes a Thread
// and returns a new Thread; nothing is mutated in place.
//
// Note: copying a Thread copies the underlying slice header but shares
// the backing array. We make a fresh slice in Reduce to avoid aliasing.
type Thread struct {
	Events []Event `json:"events"`
}

func (t Thread) Append(e Event) Thread {
	out := make([]Event, len(t.Events)+1)
	copy(out, t.Events)
	out[len(t.Events)] = e
	return Thread{Events: out}
}

func (t Thread) LastEvent() (Event, bool) {
	if len(t.Events) == 0 {
		return Event{}, false
	}
	return t.Events[len(t.Events)-1], true
}

// SerializeForLLM is the same shape as earlier chapters. The byte-exact
// stability of MarshalIndent is what makes replay tests cheap.
func (t Thread) SerializeForLLM() string {
	out, err := json.MarshalIndent(t.Events, "", "  ")
	if err != nil {
		return `{"error":"thread serialization failed"}`
	}
	return string(out)
}

// Equal compares two threads via byte-equal serialization. We use this
// in replay tests instead of reflect.DeepEqual because json.RawMessage
// has a `[]byte` field which DeepEqual handles inconsistently across
// Go versions.
func (t Thread) Equal(other Thread) bool {
	a, _ := json.Marshal(t)
	b, _ := json.Marshal(other)
	return bytes.Equal(a, b)
}
