package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
)

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

type DivideTool struct{}

func (DivideTool) Intent() string { return IntentDivide }
func (DivideTool) Execute(_ context.Context, payload json.RawMessage) (any, error) {
	var p MathPayload
	if err := json.Unmarshal(payload, &p); err != nil {
		return nil, fmt.Errorf("decode divide: %w", err)
	}
	if p.B == 0 {
		return nil, fmt.Errorf("Error: Division by zero")
	}
	return p.A / p.B, nil
}

type Registry map[string]Tool

func DefaultRegistry() Registry {
	tools := []Tool{AddTool{}, MultiplyTool{}, DivideTool{}}
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

// SafeExecute wraps Tool.Execute with a panic recover so a buggy tool
// can't crash the agent. Panics are converted to typed errors that
// follow the same self-heal path as ordinary errors.
func SafeExecute(ctx context.Context, t Tool, payload json.RawMessage) (result any, err error) {
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("tool panic in %s: %v", t.Intent(), r)
		}
	}()
	return t.Execute(ctx, payload)
}
