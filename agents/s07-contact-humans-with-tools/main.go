package main

import (
	"fmt"
	"net/http"
	"os"
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
				mathStep(IntentMultiply, 1000, 100),
				approvalStep("Send $100,000?", "high"),
				doneStep("Awaiting approval."),
			},
		},
		Registry: DefaultRegistry(),
		BaseURL:  "http://localhost" + addr,
	}
	fmt.Fprintf(os.Stderr, "listening on %s\n", addr)
	if err := http.ListenAndServe(addr, server.Handler()); err != nil {
		fmt.Fprintf(os.Stderr, "server: %v\n", err)
		os.Exit(1)
	}
}
