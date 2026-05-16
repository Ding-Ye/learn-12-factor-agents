package main

import "encoding/json"

// NextStep stays unchanged from s01/s02 — the wire format between LLM and
// agent is the one constant of the whole curriculum.
type NextStep struct {
	Intent string          `json:"intent"`
	Data   json.RawMessage `json:"data"`
}

type DoneForNowPayload struct {
	Message string `json:"message"`
}
