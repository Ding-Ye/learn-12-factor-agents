package main

import (
	"fmt"

	"github.com/Ding-Ye/learn-12-factor-agents/s10-small-focused-agents/subagents"
)

// Orchestrate runs the two-stage flow: parse user input into a calc
// plan → delegate to CalcAgent → hand the result to SummaryAgent.
//
// Note the orchestrator's thread records sub-agent boundaries
// (subagent_call / subagent_done) instead of low-level tool steps —
// staying small precisely by NOT carrying every CalcAgent step in its
// own context.
func Orchestrate(thread *Thread, userMessage string, plan subagents.CalcInput) (string, error) {
	// Delegate to CalcAgent.
	thread.Append(NewSubAgentCallEvent("CalcAgent", plan))
	calcOut, err := subagents.CalcAgent(plan)
	if err != nil {
		return "", fmt.Errorf("CalcAgent: %w", err)
	}
	thread.Append(NewSubAgentDoneEvent("CalcAgent", calcOut))

	// Delegate to SummaryAgent.
	thread.Append(NewSubAgentCallEvent("SummaryAgent", subagents.SummaryInput{
		UserMessage: userMessage,
		Result:      calcOut.Result,
	}))
	final := subagents.SummaryAgent(subagents.SummaryInput{
		UserMessage: userMessage,
		Result:      calcOut.Result,
	})
	thread.Append(NewSubAgentDoneEvent("SummaryAgent", final))

	return final, nil
}
