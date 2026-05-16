package main

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
)

func TestConsecutiveErrors_ZeroWhenNoError(t *testing.T) {
	thread := NewThread(NewUserInputEvent("hi"))
	thread.Append(NewToolCallEvent(mathStep(IntentAdd, 1, 2)))
	thread.Append(NewToolResponseEvent(3))
	if got := ConsecutiveErrors(thread); got != 0 {
		t.Errorf("got %d, want 0", got)
	}
}

func TestConsecutiveErrors_OneError(t *testing.T) {
	thread := NewThread()
	thread.Append(NewToolCallEvent(mathStep(IntentDivide, 1, 0)))
	thread.Append(NewErrorEvent("boom"))
	if got := ConsecutiveErrors(thread); got != 1 {
		t.Errorf("got %d, want 1", got)
	}
}

func TestConsecutiveErrors_ResetsAfterSuccess(t *testing.T) {
	thread := NewThread()
	thread.Append(NewToolCallEvent(mathStep(IntentDivide, 1, 0)))
	thread.Append(NewErrorEvent("e1"))
	thread.Append(NewToolCallEvent(mathStep(IntentAdd, 1, 2)))
	thread.Append(NewToolResponseEvent(3))
	if got := ConsecutiveErrors(thread); got != 0 {
		t.Errorf("got %d, want 0 (success resets streak)", got)
	}
}

func TestRunAgent_RecoversFromOneError(t *testing.T) {
	thread := NewThread(NewUserInputEvent("divide 10 by 0, then 10 by 2"))
	provider := &ScriptedSequenceProvider{
		Steps: []NextStep{
			mathStep(IntentDivide, 10, 0),
			mathStep(IntentDivide, 10, 2),
			doneStep("Recovered with 5."),
		},
	}
	final, err := RunAgent(context.Background(), thread, provider, DefaultRegistry())
	if err != nil {
		t.Fatalf("RunAgent: %v", err)
	}
	if !hasErrorEvent(final) {
		t.Errorf("expected an error event in thread")
	}
	if last, _ := final.LastEvent(); last.Type != EventTypeToolCall {
		t.Errorf("last event = %q, want tool_call", last.Type)
	}
	// Final consecutive errors should be 0 because we recovered.
	if got := ConsecutiveErrors(final); got != 0 {
		t.Errorf("ConsecutiveErrors at end = %d, want 0 after recovery", got)
	}
}

func TestRunAgent_EscalatesAfterThreeErrors(t *testing.T) {
	thread := NewThread(NewUserInputEvent("divide things by zero"))
	provider := &ScriptedSequenceProvider{
		Steps: []NextStep{
			mathStep(IntentDivide, 1, 0),
			mathStep(IntentDivide, 2, 0),
			mathStep(IntentDivide, 3, 0),
			doneStep("never reached"),
		},
	}
	_, err := RunAgent(context.Background(), thread, provider, DefaultRegistry())
	if err == nil {
		t.Fatalf("expected escalation error, got nil")
	}
	if !strings.Contains(err.Error(), "consecutive") {
		t.Errorf("error should mention 'consecutive': %v", err)
	}
}

func TestSafeExecute_RecoversFromPanic(t *testing.T) {
	_, err := SafeExecute(context.Background(), panickyTool{}, json.RawMessage(`{}`))
	if err == nil {
		t.Fatalf("expected error from panicking tool, got nil")
	}
	if !strings.Contains(err.Error(), "panic") {
		t.Errorf("error should mention 'panic': %v", err)
	}
}

func hasErrorEvent(t *Thread) bool {
	for _, e := range t.Events {
		if e.Type == EventTypeError {
			return true
		}
	}
	return false
}

type panickyTool struct{}

func (panickyTool) Intent() string { return "panicky" }
func (panickyTool) Execute(_ context.Context, _ json.RawMessage) (any, error) {
	panic("intentional test panic")
}
