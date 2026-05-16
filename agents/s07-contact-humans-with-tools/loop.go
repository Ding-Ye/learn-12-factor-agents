package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
)

const MaxSteps = 16

type Provider interface {
	DetermineNextStep(ctx context.Context, serialized string) (NextStep, error)
}

type ScriptedSequenceProvider struct {
	Steps []NextStep
	calls int
}

func (s *ScriptedSequenceProvider) DetermineNextStep(_ context.Context, _ string) (NextStep, error) {
	if s.calls >= len(s.Steps) {
		data, _ := json.Marshal(DoneForNowPayload{Message: "Sequence exhausted."})
		return NextStep{Intent: IntentDoneForNow, Data: data}, nil
	}
	step := s.Steps[s.calls]
	s.calls++
	return step, nil
}

func mathStep(intent string, a, b float64) NextStep {
	data, _ := json.Marshal(MathPayload{A: a, B: b})
	return NextStep{Intent: intent, Data: data}
}

func doneStep(message string) NextStep {
	data, _ := json.Marshal(DoneForNowPayload{Message: message})
	return NextStep{Intent: IntentDoneForNow, Data: data}
}

func approvalStep(question, stakes string) NextStep {
	data, _ := json.Marshal(RequestApprovalPayload{Question: question, Stakes: stakes})
	return NextStep{Intent: IntentRequestApproval, Data: data}
}

func clarifyStep(message string) NextStep {
	data, _ := json.Marshal(AskClarificationPayload{Message: message})
	return NextStep{Intent: IntentRequestMoreInformation, Data: data}
}

// RunAgent now treats ErrHumanContact as a clean exit (return thread,
// nil error). The HTTP layer asks AwaitingHuman() to decide whether to
// expose a response_url.
func RunAgent(ctx context.Context, thread *Thread, provider Provider, registry Registry) (*Thread, error) {
	for step := 0; step < MaxSteps; step++ {
		next, err := provider.DetermineNextStep(ctx, thread.SerializeForLLM())
		if err != nil {
			return thread, fmt.Errorf("provider at step %d: %w", step, err)
		}
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
			if errors.Is(err, ErrHumanContact) {
				// Human-contact intent: append no tool_response and
				// return cleanly so the HTTP handler can persist the
				// thread and ask for human input.
				return thread, nil
			}
			return thread, fmt.Errorf("tool %q at step %d: %w", next.Intent, step, err)
		}

		thread.Append(NewToolResponseEvent(result))
	}
	return thread, fmt.Errorf("MaxSteps=%d reached", MaxSteps)
}
