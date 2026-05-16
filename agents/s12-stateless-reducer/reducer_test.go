package main

import (
	"context"
	"testing"
)

// Replay: Reduce is referentially transparent. Same input → same
// output, forever. This is the core property the chapter is built
// around.
func TestReduce_Replay_IsByteStable(t *testing.T) {
	events := []Event{
		NewEvent(EventTypeUserInput, "add 5 and 3"),
		mathToolCall(IntentAdd, 5, 3),
		doneToolCall("8"),
	}
	a := ReduceMany(Thread{}, events)
	b := ReduceMany(Thread{}, events)
	if !a.Equal(b) {
		t.Errorf("replay produced different threads:\n a=%s\n b=%s", a.SerializeForLLM(), b.SerializeForLLM())
	}
}

func TestReduce_Replay_FromMidpoint(t *testing.T) {
	// Start the reduction from a snapshot mid-stream.
	mid := ReduceMany(Thread{}, []Event{NewEvent(EventTypeUserInput, "hi")})
	a := ReduceMany(mid, []Event{mathToolCall(IntentAdd, 1, 2), doneToolCall("3")})
	b := ReduceMany(mid, []Event{mathToolCall(IntentAdd, 1, 2), doneToolCall("3")})
	if !a.Equal(b) {
		t.Errorf("mid-stream replay diverged")
	}
}

// Fork: two divergent continuations of the same prefix produce two
// independent threads that share the prefix.
func TestReduce_Fork_SharedPrefixDivergentTails(t *testing.T) {
	prefix := []Event{
		NewEvent(EventTypeUserInput, "compute"),
		mathToolCall(IntentAdd, 1, 2),
	}
	base := ReduceMany(Thread{}, prefix)

	branchA := ReduceMany(base, []Event{mathToolCall(IntentMultiply, 3, 2), doneToolCall("6")})
	branchB := ReduceMany(base, []Event{mathToolCall(IntentMultiply, 3, 4), doneToolCall("12")})

	if branchA.Equal(branchB) {
		t.Errorf("forks with different tails should produce different threads")
	}

	// Prefixes must remain shared (no mutation back through base).
	// ReduceMany applies each event: user_input + tool_call(add); the
	// tool_call auto-step adds a tool_response, so base = 3 events.
	if len(base.Events) != 3 {
		t.Errorf("base.Events len = %d, want 3 (user_input + tool_call + auto-step tool_response)", len(base.Events))
	}
}

func TestReduce_Append_DoesNotMutateInput(t *testing.T) {
	t1 := Thread{Events: []Event{NewEvent(EventTypeUserInput, "x")}}
	t2 := t1.Append(NewEvent(EventTypeUserInput, "y"))
	// t1 must still have one event.
	if len(t1.Events) != 1 {
		t.Errorf("Append mutated input thread: len = %d", len(t1.Events))
	}
	if len(t2.Events) != 2 {
		t.Errorf("Append did not produce new thread: len = %d", len(t2.Events))
	}
}

func TestReduce_ToolCallTriggersAutoStep(t *testing.T) {
	t1 := Thread{}.Append(NewEvent(EventTypeUserInput, "go"))
	t2 := Reduce(t1, mathToolCall(IntentAdd, 5, 3))
	if len(t2.Events) != 3 {
		t.Errorf("len = %d, want 3 (user_input + tool_call + tool_response)", len(t2.Events))
	}
	if t2.Events[2].Type != EventTypeToolResponse {
		t.Errorf("event[2].Type = %q, want tool_response", t2.Events[2].Type)
	}
}

func TestRunAgent_EndToEnd(t *testing.T) {
	t1 := Thread{}.Append(NewEvent(EventTypeUserInput, "add 5 and 3, multiply by 2"))
	provider := &ScriptedSequenceProvider{
		Events: []Event{
			mathToolCall(IntentAdd, 5, 3),
			mathToolCall(IntentMultiply, 8, 2),
			doneToolCall("Result 16."),
		},
	}
	final, err := RunAgent(context.Background(), t1, provider)
	if err != nil {
		t.Fatalf("RunAgent: %v", err)
	}
	if !IsDone(final) {
		t.Errorf("final thread should be done")
	}
	// 1 user_input + 2 tool_call + 2 tool_response + 1 done tool_call = 6
	if len(final.Events) != 6 {
		t.Errorf("event count = %d, want 6", len(final.Events))
	}
}
