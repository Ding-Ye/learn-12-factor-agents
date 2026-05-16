package main

import (
	"context"
	"encoding/json"
	"fmt"
)

// Tool is the dispatch contract every concrete tool satisfies. The Intent
// method returns the literal string the provider emits as discriminator.
// Execute decodes the JSON payload and returns the result that will be
// appended to the thread as a tool_response event.
//
// We do NOT include the thread in Execute's signature — tools are pure
// (input → output). State accumulation is the loop's job, introduced in
// s05.
type Tool interface {
	Intent() string
	Execute(ctx context.Context, payload json.RawMessage) (any, error)
}

// --- concrete arithmetic tools -------------------------------------------

type AddTool struct{}

func (AddTool) Intent() string { return IntentAdd }
func (AddTool) Execute(_ context.Context, payload json.RawMessage) (any, error) {
	var p MathPayload
	if err := json.Unmarshal(payload, &p); err != nil {
		return nil, fmt.Errorf("decode add payload: %w", err)
	}
	return p.A + p.B, nil
}

type SubtractTool struct{}

func (SubtractTool) Intent() string { return IntentSubtract }
func (SubtractTool) Execute(_ context.Context, payload json.RawMessage) (any, error) {
	var p MathPayload
	if err := json.Unmarshal(payload, &p); err != nil {
		return nil, fmt.Errorf("decode subtract payload: %w", err)
	}
	return p.A - p.B, nil
}

type MultiplyTool struct{}

func (MultiplyTool) Intent() string { return IntentMultiply }
func (MultiplyTool) Execute(_ context.Context, payload json.RawMessage) (any, error) {
	var p MathPayload
	if err := json.Unmarshal(payload, &p); err != nil {
		return nil, fmt.Errorf("decode multiply payload: %w", err)
	}
	return p.A * p.B, nil
}

// DivideTool is the only arithmetic tool that can fail at execution time.
// We return the documented "Error: Division by zero" string verbatim to
// match upstream `05-agent.py:51-52`.
type DivideTool struct{}

func (DivideTool) Intent() string { return IntentDivide }
func (DivideTool) Execute(_ context.Context, payload json.RawMessage) (any, error) {
	var p MathPayload
	if err := json.Unmarshal(payload, &p); err != nil {
		return nil, fmt.Errorf("decode divide payload: %w", err)
	}
	if p.B == 0 {
		return nil, fmt.Errorf("Error: Division by zero")
	}
	return p.A / p.B, nil
}

// --- Registry: closed map of intent → Tool -------------------------------

type Registry map[string]Tool

func DefaultRegistry() Registry {
	tools := []Tool{
		AddTool{},
		SubtractTool{},
		MultiplyTool{},
		DivideTool{},
	}
	r := make(Registry, len(tools))
	for _, t := range tools {
		r[t.Intent()] = t
	}
	return r
}

// Lookup returns the tool for an intent or a typed error. Used by main's
// dispatch code so error messages cite the unknown intent literally.
func (r Registry) Lookup(intent string) (Tool, error) {
	t, ok := r[intent]
	if !ok {
		return nil, fmt.Errorf("unknown tool intent %q", intent)
	}
	return t, nil
}
