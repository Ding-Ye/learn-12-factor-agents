package main

import (
	"context"
	"testing"
)

func TestRunAgent_TwoStepThenDone(t *testing.T) {
	thread := NewThread(NewUserInputEvent("add 5 and 3, then multiply by 2"))
	provider := &ScriptedSequenceProvider{
		Steps: []NextStep{
			mathStep(IntentAdd, 5, 3),
			mathStep(IntentMultiply, 8, 2),
			doneStep("Result is 16."),
		},
	}
	final, err := RunAgent(context.Background(), thread, provider, DefaultRegistry())
	if err != nil {
		t.Fatalf("RunAgent: %v", err)
	}

	// 1 user_input + 3 tool_calls + 2 tool_responses = 6 events.
	if len(final.Events) != 6 {
		t.Errorf("event count = %d, want 6", len(final.Events))
	}
	if provider.Calls() != 3 {
		t.Errorf("provider calls = %d, want 3", provider.Calls())
	}
}

func TestIsDone_TerminatesOnDoneForNow(t *testing.T) {
	thread := NewThread(NewUserInputEvent("hi"))
	thread.Append(NewToolCallEvent(doneStep("bye")))
	if !IsDone(thread) {
		t.Errorf("IsDone should be true after appending done_for_now tool_call")
	}
}

func TestIsDone_FalseDuringRun(t *testing.T) {
	thread := NewThread(NewUserInputEvent("hi"))
	thread.Append(NewToolCallEvent(mathStep(IntentAdd, 1, 2)))
	thread.Append(NewToolResponseEvent(3))
	if IsDone(thread) {
		t.Errorf("IsDone should be false mid-loop")
	}
}

func TestLastToolCall(t *testing.T) {
	thread := NewThread(NewUserInputEvent("hi"))
	thread.Append(NewToolCallEvent(mathStep(IntentAdd, 1, 2)))
	thread.Append(NewToolResponseEvent(3))
	thread.Append(NewToolCallEvent(mathStep(IntentMultiply, 3, 4)))

	step, ok := LastToolCall(thread)
	if !ok {
		t.Fatalf("LastToolCall returned ok=false")
	}
	if step.Intent != IntentMultiply {
		t.Errorf("LastToolCall.Intent = %q, want %q", step.Intent, IntentMultiply)
	}
}

func TestRunAgent_DispatchErrorPropagates(t *testing.T) {
	thread := NewThread(NewUserInputEvent("hi"))
	provider := &ScriptedSequenceProvider{
		Steps: []NextStep{
			{Intent: "no_such_tool", Data: []byte(`{}`)},
		},
	}
	_, err := RunAgent(context.Background(), thread, provider, DefaultRegistry())
	if err == nil {
		t.Fatalf("expected dispatch error, got nil")
	}
}

func TestRunAgent_MaxStepsBreak(t *testing.T) {
	// Provider that never returns done_for_now and never makes progress
	// (math tool that no-ops). The loop should bail at MaxSteps with an
	// error mentioning the limit.
	thread := NewThread(NewUserInputEvent("hi"))
	steps := make([]NextStep, MaxSteps+5)
	for i := range steps {
		steps[i] = mathStep(IntentAdd, 1, 1)
	}
	provider := &ScriptedSequenceProvider{Steps: steps}
	_, err := RunAgent(context.Background(), thread, provider, DefaultRegistry())
	if err == nil {
		t.Fatalf("expected MaxSteps error, got nil")
	}
}
