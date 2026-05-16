package main

import (
	"context"
	"fmt"
	"os"
)

func main() {
	t := Thread{}.Append(NewEvent(EventTypeUserInput, "add 5 and 3, then multiply by 2"))
	provider := &ScriptedSequenceProvider{
		Events: []Event{
			mathToolCall(IntentAdd, 5, 3),
			mathToolCall(IntentMultiply, 8, 2),
			doneToolCall("Result is 16."),
		},
	}
	final, err := RunAgent(context.Background(), t, provider)
	if err != nil {
		fmt.Fprintf(os.Stderr, "RunAgent: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("Final thread (%d events):\n", len(final.Events))
	for i, e := range final.Events {
		fmt.Printf("  [%d] %s: %s\n", i, e.Type, string(e.Data))
	}
}
