package main

import (
	"strings"
	"testing"

	"github.com/Ding-Ye/learn-12-factor-agents/s10-small-focused-agents/subagents"
)

func TestOrchestrate_TwoStageFlow(t *testing.T) {
	msg := "add 5 and 3, then multiply by 2"
	thread := NewThread(NewUserInputEvent(msg))
	plan := subagents.CalcInput{
		Steps: []subagents.CalcStep{
			{Op: "add", A: 5, B: 3},
			{Op: "multiply", B: 2},
		},
	}
	final, err := Orchestrate(thread, msg, plan)
	if err != nil {
		t.Fatalf("Orchestrate: %v", err)
	}
	if !strings.Contains(final, "16") {
		t.Errorf("final summary should mention 16: %q", final)
	}
}

func TestOrchestrate_ThreadRecordsSubAgentBoundaries(t *testing.T) {
	thread := NewThread(NewUserInputEvent("compute"))
	plan := subagents.CalcInput{Steps: []subagents.CalcStep{{Op: "add", A: 1, B: 2}}}
	_, err := Orchestrate(thread, "compute", plan)
	if err != nil {
		t.Fatalf("Orchestrate: %v", err)
	}
	// 1 user_input + 2 subagent_call + 2 subagent_done = 5 events.
	if len(thread.Events) != 5 {
		t.Errorf("event count = %d, want 5", len(thread.Events))
	}
	for i, want := range []string{
		EventTypeUserInput,
		EventTypeSubAgentCall,
		EventTypeSubAgentDone,
		EventTypeSubAgentCall,
		EventTypeSubAgentDone,
	} {
		if thread.Events[i].Type != want {
			t.Errorf("event[%d].Type = %q, want %q", i, thread.Events[i].Type, want)
		}
	}
}

func TestCalcAgent_ChainsResults(t *testing.T) {
	out, err := subagents.CalcAgent(subagents.CalcInput{
		Steps: []subagents.CalcStep{
			{Op: "add", A: 5, B: 3},
			{Op: "multiply", B: 2}, // 8*2=16
			{Op: "add", B: 4},      // 16+4=20
		},
	})
	if err != nil {
		t.Fatalf("CalcAgent: %v", err)
	}
	if out.Result != 20 {
		t.Errorf("result = %v, want 20", out.Result)
	}
	if len(out.Trace) != 3 {
		t.Errorf("trace len = %d, want 3", len(out.Trace))
	}
}

func TestCalcAgent_UnknownOp(t *testing.T) {
	_, err := subagents.CalcAgent(subagents.CalcInput{
		Steps: []subagents.CalcStep{{Op: "exponentiate", A: 2, B: 8}},
	})
	if err == nil {
		t.Fatalf("expected error for unknown op, got nil")
	}
}

func TestSummaryAgent_TemplatesResult(t *testing.T) {
	got := subagents.SummaryAgent(subagents.SummaryInput{
		UserMessage: "do math",
		Result:      42,
	})
	if !strings.Contains(got, "42") {
		t.Errorf("summary missing result 42: %q", got)
	}
	if !strings.Contains(got, "do math") {
		t.Errorf("summary missing user message: %q", got)
	}
}

func TestOrchestrate_ThreadsAreIndependent(t *testing.T) {
	// CalcAgent's input/output don't sit in the orchestrator's
	// per-step trace. We only see boundary events. This test asserts
	// the orchestrator's thread DOES NOT contain a tool_call for the
	// internal math ops.
	thread := NewThread(NewUserInputEvent("compute"))
	plan := subagents.CalcInput{Steps: []subagents.CalcStep{
		{Op: "add", A: 1, B: 2},
		{Op: "multiply", B: 3},
	}}
	_, _ = Orchestrate(thread, "compute", plan)
	for _, e := range thread.Events {
		if e.Type == EventTypeToolCall || e.Type == EventTypeToolResponse {
			t.Errorf("orchestrator thread should not contain low-level tool_call/tool_response: %+v", e)
		}
	}
}
