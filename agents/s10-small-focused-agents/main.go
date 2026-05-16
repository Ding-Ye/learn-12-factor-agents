package main

import (
	"fmt"
	"os"

	"github.com/Ding-Ye/learn-12-factor-agents/s10-small-focused-agents/subagents"
)

func main() {
	msg := "add 5 and 3, then multiply by 2"
	thread := NewThread(NewUserInputEvent(msg))

	plan := subagents.CalcInput{
		Steps: []subagents.CalcStep{
			{Op: "add", A: 5, B: 3},
			{Op: "multiply", B: 2}, // A is filled from previous result
		},
	}

	final, err := Orchestrate(thread, msg, plan)
	if err != nil {
		fmt.Fprintf(os.Stderr, "orchestrate: %v\n", err)
		os.Exit(1)
	}
	fmt.Println(final)

	fmt.Println("\nOrchestrator thread:")
	for i, e := range thread.Events {
		fmt.Printf("  [%d] %s\n", i, e.Type)
	}
}
