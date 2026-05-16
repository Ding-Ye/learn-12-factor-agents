package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
)

// ErrHumanContact signals that the loop must STOP and return the thread
// to the caller; the tool itself doesn't execute. RunAgent checks for
// this error specifically (s07 addition).
var ErrHumanContact = errors.New("human contact requested")

type Tool interface {
	Intent() string
	Execute(ctx context.Context, payload json.RawMessage) (any, error)
}

type AddTool struct{}

func (AddTool) Intent() string { return IntentAdd }
func (AddTool) Execute(_ context.Context, payload json.RawMessage) (any, error) {
	var p MathPayload
	if err := json.Unmarshal(payload, &p); err != nil {
		return nil, fmt.Errorf("decode add: %w", err)
	}
	return p.A + p.B, nil
}

type MultiplyTool struct{}

func (MultiplyTool) Intent() string { return IntentMultiply }
func (MultiplyTool) Execute(_ context.Context, payload json.RawMessage) (any, error) {
	var p MathPayload
	if err := json.Unmarshal(payload, &p); err != nil {
		return nil, fmt.Errorf("decode multiply: %w", err)
	}
	return p.A * p.B, nil
}

// RequestApprovalTool and AskClarificationTool both return
// `ErrHumanContact` from Execute. They never actually compute anything
// — the "execution" is the human's eventual reply via POST /response.
type RequestApprovalTool struct{}

func (RequestApprovalTool) Intent() string { return IntentRequestApproval }
func (RequestApprovalTool) Execute(_ context.Context, _ json.RawMessage) (any, error) {
	return nil, ErrHumanContact
}

type AskClarificationTool struct{}

func (AskClarificationTool) Intent() string { return IntentRequestMoreInformation }
func (AskClarificationTool) Execute(_ context.Context, _ json.RawMessage) (any, error) {
	return nil, ErrHumanContact
}

type Registry map[string]Tool

func DefaultRegistry() Registry {
	tools := []Tool{
		AddTool{},
		MultiplyTool{},
		RequestApprovalTool{},
		AskClarificationTool{},
	}
	r := make(Registry, len(tools))
	for _, t := range tools {
		r[t.Intent()] = t
	}
	return r
}

func (r Registry) Lookup(intent string) (Tool, error) {
	t, ok := r[intent]
	if !ok {
		return nil, fmt.Errorf("unknown tool intent %q", intent)
	}
	return t, nil
}
