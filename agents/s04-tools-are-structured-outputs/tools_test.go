package main

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
)

func TestRegistry_LookupKnownIntents(t *testing.T) {
	r := DefaultRegistry()
	for _, intent := range []string{IntentAdd, IntentSubtract, IntentMultiply, IntentDivide} {
		tool, err := r.Lookup(intent)
		if err != nil {
			t.Errorf("Lookup(%q): %v", intent, err)
			continue
		}
		if tool.Intent() != intent {
			t.Errorf("Lookup(%q) returned tool with Intent()=%q", intent, tool.Intent())
		}
	}
}

func TestRegistry_LookupUnknown(t *testing.T) {
	_, err := DefaultRegistry().Lookup("definitely_not_a_tool")
	if err == nil {
		t.Fatalf("expected error for unknown intent, got nil")
	}
	if !strings.Contains(err.Error(), "definitely_not_a_tool") {
		t.Errorf("error should cite the bad intent: %v", err)
	}
}

func TestAddTool_Execute(t *testing.T) {
	payload, _ := json.Marshal(MathPayload{A: 2, B: 3})
	got, err := AddTool{}.Execute(context.Background(), payload)
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if got != 5.0 {
		t.Errorf("got %v, want 5", got)
	}
}

func TestDivideTool_DivideByZero(t *testing.T) {
	payload, _ := json.Marshal(MathPayload{A: 5, B: 0})
	_, err := DivideTool{}.Execute(context.Background(), payload)
	if err == nil {
		t.Fatalf("expected divide-by-zero error, got nil")
	}
	if !strings.Contains(err.Error(), "Division by zero") {
		t.Errorf("error should say 'Division by zero': %v", err)
	}
}

func TestMultiplyTool_Execute(t *testing.T) {
	payload, _ := json.Marshal(MathPayload{A: 4, B: 6})
	got, err := MultiplyTool{}.Execute(context.Background(), payload)
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if got != 24.0 {
		t.Errorf("got %v, want 24", got)
	}
}

func TestScriptedProvider_RoutesByInput(t *testing.T) {
	cases := []struct {
		in     string
		intent string
	}{
		{"add two and three", IntentAdd},
		{"can you subtract 5 from 10", IntentSubtract},
		{"multiply 4 and 6", IntentMultiply},
		{"divide 10 by 2", IntentDivide},
		{"just say hi", IntentDoneForNow},
	}
	for _, c := range cases {
		step, err := ScriptedProvider{}.DetermineNextStep(context.Background(), c.in)
		if err != nil {
			t.Errorf("input %q: %v", c.in, err)
			continue
		}
		if step.Intent != c.intent {
			t.Errorf("input %q: intent=%q, want %q", c.in, step.Intent, c.intent)
		}
	}
}

func TestNextStep_PayloadSurvivesRoundTrip(t *testing.T) {
	original, _ := ScriptedProvider{}.DetermineNextStep(context.Background(), "add 1 and 2")
	encoded, err := json.Marshal(original)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	var decoded NextStep
	if err := json.Unmarshal(encoded, &decoded); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	var p MathPayload
	if err := json.Unmarshal(decoded.Data, &p); err != nil {
		t.Fatalf("decode payload: %v", err)
	}
	if p.A != 2 || p.B != 3 {
		t.Errorf("payload mismatch: a=%v b=%v, want a=2 b=3", p.A, p.B)
	}
}
