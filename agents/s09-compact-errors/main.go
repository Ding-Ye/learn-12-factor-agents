package main

import (
	"context"
	"fmt"
	"os"
)

func main() {
	thread := NewThread(NewUserInputEvent("divide 10 by 0, then 10 by 2"))
	provider := &ScriptedSequenceProvider{
		Steps: []NextStep{
			mathStep(IntentDivide, 10, 0), // will fail
			mathStep(IntentDivide, 10, 2), // succeeds (5)
			doneStep("Recovered with 5."),
		},
	}
	final, err := RunAgent(context.Background(), thread, provider, DefaultRegistry())
	if err != nil {
		fmt.Fprintf(os.Stderr, "RunAgent: %v\n", err)
	}
	fmt.Printf("Final thread has %d events:\n", len(final.Events))
	for i, e := range final.Events {
		fmt.Printf("  [%d] %s: %v\n", i, e.Type, eventSummary(e))
	}
	fmt.Printf("Consecutive errors at end: %d\n", ConsecutiveErrors(final))
}

func eventSummary(e Event) string {
	switch v := e.Data.(type) {
	case NextStep:
		return "intent=" + v.Intent
	default:
		return fmt.Sprintf("%v", v)
	}
}
