package main

import (
	"context"
	"fmt"
	"os"
)

func main() {
	thread := NewThread(NewUserInputEvent("add 5 and 3, then multiply by 2, please approve"))
	provider := &ScriptedSequenceProvider{
		Steps: []NextStep{
			mathStep(IntentAdd, 5, 3),
			mathStep(IntentMultiply, 8, 2),
			approvalStep("Send result 16?", "low"),
			doneStep("approved, sent"),
		},
	}
	final, err := RunAgent(context.Background(), thread, provider, DefaultRegistry())
	if err != nil {
		fmt.Fprintf(os.Stderr, "RunAgent: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("Final thread has %d events:\n", len(final.Events))
	for i, e := range final.Events {
		fmt.Printf("  [%d] %s\n", i, e.Type)
	}
}
