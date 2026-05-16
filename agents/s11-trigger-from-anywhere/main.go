package main

import (
	"fmt"
	"net/http"
	"os"

	"github.com/Ding-Ye/learn-12-factor-agents/s11-trigger-from-anywhere/triggers"
)

func main() {
	addr := ":8080"
	if len(os.Args) > 1 {
		addr = os.Args[1]
	}
	server := &Server{
		Store: NewMemoryStore(),
		Provider: &ScriptedSequenceProvider{
			Steps: []NextStep{
				mathStep(2, 3),
				doneStep("Computed 5."),
			},
		},
		Triggers: map[string]triggers.Trigger{
			"slack":   triggers.SlackTrigger{},
			"webhook": triggers.WebhookTrigger{},
		},
	}
	fmt.Fprintf(os.Stderr, "listening on %s\n", addr)
	if err := http.ListenAndServe(addr, server.Handler()); err != nil {
		fmt.Fprintf(os.Stderr, "server: %v\n", err)
		os.Exit(1)
	}
}
