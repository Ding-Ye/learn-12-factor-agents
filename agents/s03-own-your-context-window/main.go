package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"
)

func main() {
	userInput := strings.Join(os.Args[1:], " ")
	if userInput == "" {
		userInput = "hello"
	}

	// Seed the thread with one user_input event.
	thread := NewThread(NewUserInputEvent(userInput))

	rendered, err := RenderPrompt(PromptInput{Thread: thread.SerializeForLLM()})
	if err != nil {
		fmt.Fprintf(os.Stderr, "render: %v\n", err)
		os.Exit(1)
	}

	step, err := EchoThreadProvider{}.DetermineNextStep(context.Background(), rendered)
	if err != nil {
		fmt.Fprintf(os.Stderr, "provider: %v\n", err)
		os.Exit(1)
	}

	out, err := renderNextStep(step)
	if err != nil {
		fmt.Fprintf(os.Stderr, "format: %v\n", err)
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
