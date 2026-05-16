package main

import (
	"context"
	"fmt"
)

// MaxSteps caps the loop so a misbehaving provider can't spin forever.
// Real systems would do this with context.WithTimeout; we keep it
// declarative for teaching purposes.
const MaxSteps = 16

// RunAgent is the s05 agent loop. It pulls the next step from the
// provider, looks up the matching tool, executes it, appends both the
// tool_call and tool_response events to the thread, and repeats until
// the provider returns done_for_now (or MaxSteps is reached).
//
// Notice what RunAgent does NOT do:
//   - It doesn't keep any state outside the Thread.
//   - It doesn't reach back to the CLI or HTTP.
//   - It returns the same Thread it was given (mutated) plus an error.
//
// This is the smallest version of factor-05 in code.
func RunAgent(ctx context.Context, thread *Thread, provider Provider, registry Registry) (*Thread, error) {
	for step := 0; step < MaxSteps; step++ {
		next, err := provider.DetermineNextStep(ctx, thread.SerializeForLLM())
		if err != nil {
			return thread, fmt.Errorf("provider call at step %d: %w", step, err)
		}

		// Always append the tool_call event — even for done_for_now —
		// so the thread is a complete record of decisions taken.
		thread.Append(NewToolCallEvent(next))

		if next.Intent == IntentDoneForNow {
			return thread, nil
		}

		tool, err := registry.Lookup(next.Intent)
		if err != nil {
			return thread, fmt.Errorf("dispatch at step %d: %w", step, err)
		}

		result, err := tool.Execute(ctx, next.Data)
		if err != nil {
			// s09 makes this an event-on-thread instead. For s05 we
			// surface the error directly so tests can see it.
			return thread, fmt.Errorf("tool %q at step %d: %w", next.Intent, step, err)
		}

		thread.Append(NewToolResponseEvent(result))
	}
	return thread, fmt.Errorf("agent loop hit MaxSteps=%d without reaching done_for_now", MaxSteps)
}
