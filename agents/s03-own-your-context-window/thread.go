package main

import (
	"encoding/json"
)

// Thread is the append-only event log. By convention we never mutate past
// events; we only `Append` new ones. The Events slice is exported so tests
// can read it directly, but writers should go through Append.
//
// Upstream Thread (workshops/2025-07-16/walkthrough/01-agent.py:12-19) is
// almost identical — a list with one method, `serialize_for_llm`. We add
// `LastEvent` so later chapters can detect "is the agent waiting?"
// without poking at the slice directly.
type Thread struct {
	Events []Event
}

// NewThread returns a thread seeded with the given events (commonly just
// a user_input event). Always seed via the constructor — empty threads
// confuse the downstream rendering tests.
func NewThread(seed ...Event) *Thread {
	t := &Thread{Events: make([]Event, 0, 8)}
	t.Events = append(t.Events, seed...)
	return t
}

// Append adds an event to the end of the log. Pointer receiver so callers
// don't accidentally lose appends.
func (t *Thread) Append(e Event) {
	t.Events = append(t.Events, e)
}

// LastEvent returns the most recently appended event and whether one
// exists. Used in s05+ to ask "are we waiting on a tool / human?"
func (t *Thread) LastEvent() (Event, bool) {
	if len(t.Events) == 0 {
		return Event{}, false
	}
	return t.Events[len(t.Events)-1], true
}

// SerializeForLLM converts the thread into the string the Provider will
// see. JSON is the chapter-3 default; s07 demonstrates an XML variant
// that's cheaper in tokens. We use MarshalIndent so test-failure diffs
// are readable.
func (t *Thread) SerializeForLLM() string {
	out, err := json.MarshalIndent(t.Events, "", "  ")
	if err != nil {
		// json.Marshal of `[]Event` can fail if Event.Data contains an
		// unencodable value (channels, funcs). All constructors above
		// produce JSON-safe payloads, so this is a programmer error.
		// We surface it as a placeholder string rather than panicking
		// so tests still get a useful diff if it ever happens.
		return `{"error": "thread serialization failed"}`
	}
	return string(out)
}
