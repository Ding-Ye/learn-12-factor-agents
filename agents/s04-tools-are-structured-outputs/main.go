package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"
)

// main performs ONE dispatch step: ask the provider, look up the tool by
// intent, execute it once, print the result. s05 adds the loop.
func main() {
	userInput := strings.Join(os.Args[1:], " ")
	if userInput == "" {
		userInput = "add 2 and 3"
	}

	step, err := ScriptedProvider{}.DetermineNextStep(context.Background(), userInput)
	if err != nil {
		fmt.Fprintf(os.Stderr, "provider: %v\n", err)
		os.Exit(1)
	}

	registry := DefaultRegistry()

	// done_for_now isn't a Tool; handle it inline.
	if step.Intent == IntentDoneForNow {
		var p DoneForNowPayload
		_ = json.Unmarshal(step.Data, &p)
		fmt.Printf("intent=%s message=%q\n", step.Intent, p.Message)
		return
	}

	tool, err := registry.Lookup(step.Intent)
	if err != nil {
		fmt.Fprintf(os.Stderr, "dispatch: %v\n", err)
		os.Exit(1)
	}

	result, err := tool.Execute(context.Background(), step.Data)
	if err != nil {
		fmt.Fprintf(os.Stderr, "tool error: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("intent=%s payload=%s result=%v\n", step.Intent, string(step.Data), result)
}
