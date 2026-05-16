package main

import (
	"context"
	"encoding/json"
	"fmt"
)

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

type Registry map[string]Tool

func DefaultRegistry() Registry {
	tools := []Tool{AddTool{}, MultiplyTool{}}
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
