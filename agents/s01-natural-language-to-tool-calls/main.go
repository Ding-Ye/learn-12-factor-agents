package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"
)

// main wires a CLI front-end to the EchoProvider. The CLI joins all argv into
// a single user message, asks the provider for the next step, and prints the
// intent + decoded payload.
//
// Usage:
//
//	go run . "hello"
//	→ intent=done_for_now message="Hello! How can I assist you today?"
func main() {
	userInput := strings.Join(os.Args[1:], " ")
	if userInput == "" {
		userInput = "hello"
	}

	provider := EchoProvider{}
	step, err := provider.DetermineNextStep(context.Background(), userInput)
	if err != nil {
		fmt.Fprintf(os.Stderr, "provider error: %v\n", err)
		os.Exit(1)
	}

	// Decode the payload so we can print it in a human-friendly way. The
	// curriculum's later chapters dispatch on Intent here; in s01 we just
	// print.
	out, err := renderNextStep(step)
	if err != nil {
		fmt.Fprintf(os.Stderr, "render error: %v\n", err)
		os.Exit(1)
	}
	fmt.Println(out)
}

// renderNextStep is split out so tests can exercise the formatting without
// going through os.Args / os.Exit.
func renderNextStep(step NextStep) (string, error) {
	switch step.Intent {
	case "done_for_now":
		var payload DoneForNowPayload
		if err := json.Unmarshal(step.Data, &payload); err != nil {
			return "", fmt.Errorf("decode done_for_now: %w", err)
		}
		return fmt.Sprintf("intent=%s message=%q", step.Intent, payload.Message), nil
	default:
		// Unknown intents will appear from s04 onward when the curriculum
		// introduces real tool calls. For now, fall back to a generic dump
		// so the program still terminates cleanly.
		return fmt.Sprintf("intent=%s data=%s", step.Intent, string(step.Data)), nil
	}
}
