package main

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
)

func TestThread_AppendPreservesOrder(t *testing.T) {
	thread := NewThread(NewUserInputEvent("hi"))
	thread.Append(NewToolCallEvent(map[string]any{"intent": "add", "a": 2, "b": 3}))
	thread.Append(NewToolResponseEvent(5))

	if len(thread.Events) != 3 {
		t.Fatalf("len(events) = %d, want 3", len(thread.Events))
	}
	wantTypes := []string{EventTypeUserInput, EventTypeToolCall, EventTypeToolResponse}
	for i, want := range wantTypes {
		if thread.Events[i].Type != want {
			t.Errorf("event[%d].Type = %q, want %q", i, thread.Events[i].Type, want)
		}
	}
}

func TestThread_SerializeForLLM_StableAcrossRuns(t *testing.T) {
	build := func() *Thread {
		thread := NewThread(NewUserInputEvent("ping"))
		thread.Append(NewToolCallEvent(map[string]any{"x": 1}))
		return thread
	}
	a := build().SerializeForLLM()
	b := build().SerializeForLLM()
	if a != b {
		t.Errorf("serialization not byte-stable across builds:\n a=%s\n b=%s", a, b)
	}
}

func TestThread_LastEvent(t *testing.T) {
	thread := NewThread()
	if _, ok := thread.LastEvent(); ok {
		t.Errorf("LastEvent on empty thread returned ok=true")
	}
	thread.Append(NewUserInputEvent("x"))
	last, ok := thread.LastEvent()
	if !ok {
		t.Fatalf("LastEvent on non-empty thread returned ok=false")
	}
	if last.Type != EventTypeUserInput {
		t.Errorf("LastEvent.Type = %q, want %q", last.Type, EventTypeUserInput)
	}
}

func TestRenderPrompt_ContainsSerializedThread(t *testing.T) {
	thread := NewThread(NewUserInputEvent("add 5 and 3"))
	rendered, err := RenderPrompt(PromptInput{Thread: thread.SerializeForLLM()})
	if err != nil {
		t.Fatalf("RenderPrompt: %v", err)
	}
	if !strings.Contains(rendered, `"type": "user_input"`) {
		t.Errorf("rendered prompt missing user_input event:\n%s", rendered)
	}
	if !strings.Contains(rendered, "add 5 and 3") {
		t.Errorf("rendered prompt missing user message:\n%s", rendered)
	}
}

func TestEchoThreadProvider_AcknowledgesEventCount(t *testing.T) {
	thread := NewThread(NewUserInputEvent("hi"))
	rendered, _ := RenderPrompt(PromptInput{Thread: thread.SerializeForLLM()})
	step, err := EchoThreadProvider{}.DetermineNextStep(context.Background(), rendered)
	if err != nil {
		t.Fatalf("provider: %v", err)
	}
	var p DoneForNowPayload
	if err := json.Unmarshal(step.Data, &p); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if !strings.Contains(p.Message, "1 user_input event") {
		t.Errorf("message = %q, want it to mention '1 user_input event'", p.Message)
	}
}

func TestEvent_JSONRoundTrip_TypeStable(t *testing.T) {
	original := NewUserInputEvent("hello")
	b, err := json.Marshal(original)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	var decoded Event
	if err := json.Unmarshal(b, &decoded); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if decoded.Type != original.Type {
		t.Errorf("type mismatch after round trip: got %q, want %q", decoded.Type, original.Type)
	}
	// Note: decoded.Data is now `interface{}` carrying a string for this
	// simple case. We document this widening in s12.
	if s, ok := decoded.Data.(string); !ok || s != "hello" {
		t.Errorf("data mismatch: got %v (type %T), want \"hello\" (string)", decoded.Data, decoded.Data)
	}
}
