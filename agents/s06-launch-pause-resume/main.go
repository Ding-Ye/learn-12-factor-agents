package main

import (
	"fmt"
	"net/http"
	"os"
)

// main spins up the HTTP server on :8080 with a deterministic scripted
// provider. Try:
//
//   go run . &
//   curl -X POST localhost:8080/thread -d '{"message":"add 5 and 3 then multiply by 2"}'
//   curl localhost:8080/thread/<id>
func main() {
	addr := ":8080"
	if len(os.Args) > 1 {
		addr = os.Args[1]
	}

	server := &Server{
		Store: NewMemoryStore(),
		Provider: &ScriptedSequenceProvider{
			Steps: []NextStep{
				mathStep(IntentAdd, 5, 3),
				mathStep(IntentMultiply, 8, 2),
				doneStep("Result is 16."),
			},
		},
		Registry: DefaultRegistry(),
	}

	fmt.Fprintf(os.Stderr, "listening on %s\n", addr)
	if err := http.ListenAndServe(addr, server.Handler()); err != nil {
		fmt.Fprintf(os.Stderr, "server: %v\n", err)
		os.Exit(1)
	}
}
