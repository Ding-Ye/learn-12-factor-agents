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
