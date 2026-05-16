package main

import (
	"context"
	"testing"
)

func TestControlFlow_AllIntentsCovered(t *testing.T) {
	// Every intent in the closed set must produce a non-Invalid Action.
	registry := DefaultRegistry()
	for _, intent := range KnownIntents() {
		got := ControlFlow(NewThread(), NextStep{Intent: intent}, registry)
		if got == ActionInvalid {
			t.Errorf("ControlFlow returned ActionInvalid for known intent %q", intent)
		}
	}
}

func TestControlFlow_DoneForNow_IsFinish(t *testing.T) {
	if got := ControlFlow(NewThread(), NextStep{Intent: IntentDoneForNow}, DefaultRegistry()); got != ActionFinish {
		t.Errorf("got %v, want ActionFinish", got)
	}
}

func TestControlFlow_HumanIntents_AreBreak(t *testing.T) {
	for _, intent := range []string{IntentRequestApproval, IntentRequestMoreInformation} {
		got := ControlFlow(NewThread(), NextStep{Intent: intent}, DefaultRegistry())
		if got != ActionBreak {
			t.Errorf("intent %q: got %v, want ActionBreak", intent, got)
		}
	}
}

func TestControlFlow_MathIntents_AreLoop(t *testing.T) {
	for _, intent := range []string{IntentAdd, IntentMultiply} {
		got := ControlFlow(NewThread(), NextStep{Intent: intent}, DefaultRegistry())
		if got != ActionLoop {
			t.Errorf("intent %q: got %v, want ActionLoop", intent, got)
		}
	}
}

func TestControlFlow_UnknownIntent_Escalates(t *testing.T) {
	got := ControlFlow(NewThread(), NextStep{Intent: "ride_a_bike"}, DefaultRegistry())
	if got != ActionEscalate {
		t.Errorf("got %v, want ActionEscalate", got)
	}
}

func TestRunAgent_Math_Then_Approval_Breaks(t *testing.T) {
	thread := NewThread(NewUserInputEvent("hi"))
	provider := &ScriptedSequenceProvider{
		Steps: []NextStep{
			mathStep(IntentAdd, 1, 2),
			approvalStep("ok?", "low"),
			doneStep("never reached"),
		},
	}
	final, err := RunAgent(context.Background(), thread, provider, DefaultRegistry())
	if err != nil {
		t.Fatalf("RunAgent: %v", err)
	}
	if provider.calls != 2 {
		t.Errorf("provider should be called exactly twice; got %d", provider.calls)
	}
	if got, _ := final.LastEvent(); got.Type != EventTypeToolCall {
		t.Errorf("last event = %q, want tool_call", got.Type)
	}
}

func TestRunAgent_UnknownIntent_AppendsErrorEvent(t *testing.T) {
	thread := NewThread(NewUserInputEvent("hi"))
	provider := &ScriptedSequenceProvider{
		Steps: []NextStep{{Intent: "ride_a_bike", Data: []byte(`{}`)}},
	}
	_, err := RunAgent(context.Background(), thread, provider, DefaultRegistry())
	if err == nil {
		t.Fatalf("expected escalate error, got nil")
	}
	if !hasErrorEvent(thread) {
		t.Errorf("expected error event in thread; got %+v", thread.Events)
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
