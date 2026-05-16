package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"
)

// main runs a 3-step demo: add(5, 3) → multiply(8, 2) → done_for_now.
// The provider is scripted so the demo is reproducible without an LLM.
func main() {
	userInput := strings.Join(os.Args[1:], " ")
	if userInput == "" {
		userInput = "add 5 and 3, then multiply by 2"
	}

	thread := NewThread(NewUserInputEvent(userInput))

	provider := &ScriptedSequenceProvider{
		Steps: []NextStep{
			mathStep(IntentAdd, 5, 3),
			mathStep(IntentMultiply, 8, 2),
			doneStep("The result is 16."),
		},
	}

	registry := DefaultRegistry()

	final, err := RunAgent(context.Background(), thread, provider, registry)
	if err != nil {
		fmt.Fprintf(os.Stderr, "RunAgent: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Loop ran %d turns. Final thread has %d events.\n", provider.Calls(), len(final.Events))
	for i, e := range final.Events {
		switch v := e.Data.(type) {
		case NextStep:
			fmt.Printf("  [%d] %s: intent=%s\n", i, e.Type, v.Intent)
		default:
			b, _ := json.Marshal(v)
			fmt.Printf("  [%d] %s: %s\n", i, e.Type, string(b))
		}
	}
}
