package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"
)

// main wires CLI → RenderPrompt → RecordingProvider → renderNextStep.
// Compared to s01, the rendered prompt now sits between argv and the
// provider, and the provider's output explicitly references the prompt
// hash so a reader can see the round-trip working.
func main() {
	userInput := strings.Join(os.Args[1:], " ")
	if userInput == "" {
		userInput = "hello"
	}

	rendered, err := RenderPrompt(PromptInput{UserInput: userInput})
	if err != nil {
		fmt.Fprintf(os.Stderr, "render error: %v\n", err)
		os.Exit(1)
	}

	provider := &RecordingProvider{}
	step, err := provider.DetermineNextStep(context.Background(), rendered)
	if err != nil {
		fmt.Fprintf(os.Stderr, "provider error: %v\n", err)
		os.Exit(1)
	}

	out, err := renderNextStep(step)
	if err != nil {
		fmt.Fprintf(os.Stderr, "format error: %v\n", err)
		os.Exit(1)
	}
	fmt.Println(out)
}

func renderNextStep(step NextStep) (string, error) {
	switch step.Intent {
	case "done_for_now":
		var p DoneForNowPayload
		if err := json.Unmarshal(step.Data, &p); err != nil {
			return "", fmt.Errorf("decode done_for_now: %w", err)
		}
		return fmt.Sprintf("intent=%s message=%q", step.Intent, p.Message), nil
	default:
		return fmt.Sprintf("intent=%s data=%s", step.Intent, string(step.Data)), nil
	}
}
