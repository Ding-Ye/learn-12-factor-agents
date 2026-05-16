package main

import (
	"bytes"
	"fmt"
	"text/template"
)

// promptTemplate now substitutes the serialized Thread instead of a raw
// UserInput. Compared to s02 only the variable name changed; the LLM
// still sees a SYSTEM: / USER: split.
const promptTemplate = `SYSTEM:
You are a helpful assistant that responds to the user's message.

USER:
You are given the following thread of events:
{{ .Thread }}

What should the next step be?
`

// PromptInput is the parameter struct for RenderPrompt. The single field
// is the JSON-serialized thread coming from Thread.SerializeForLLM.
type PromptInput struct {
	Thread string
}

func RenderPrompt(in PromptInput) (string, error) {
	t, err := template.New("agent-prompt").Parse(promptTemplate)
	if err != nil {
		return "", fmt.Errorf("parse template: %w", err)
	}
	var buf bytes.Buffer
	if err := t.Execute(&buf, in); err != nil {
		return "", fmt.Errorf("execute template: %w", err)
	}
	return buf.String(), nil
}
