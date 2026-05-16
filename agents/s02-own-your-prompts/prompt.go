package main

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"text/template"
)

// promptTemplate mirrors the structure of the upstream BAML prompt block at
// workshops/2025-07-16/walkthrough/01-agent.baml lines 12-26:
//
//   {{ _.role("system") }}
//   You are a helpful assistant that responds to the user's message.
//   {{ _.role("user") }}
//   You are given the following thread of events:
//   {{ thread }}
//
// We don't yet have a Thread (that's s03), so the template plugs the raw
// user input into the user-role section. Two role markers (SYSTEM:, USER:)
// keep the structure visible in tests.
const promptTemplate = `SYSTEM:
You are a helpful assistant that responds to the user's message.

USER:
You are given the following thread of events:
{{ .UserInput }}

What should the next step be?
`

// PromptInput carries everything the template needs. Keeping it as a struct
// (rather than a free map) means new template fields show up as compile
// errors, not silent missing data.
type PromptInput struct {
	UserInput string
}

// RenderPrompt instantiates promptTemplate with the given input. The
// returned string is exactly what the Provider will see — no hidden
// framework rewrites. Errors only surface if the template fails to parse,
// which is a programmer error caught by tests.
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

// PromptHash gives a short stable fingerprint of a rendered prompt. The
// RecordingProvider echoes this back so tests can assert the rendered
// prompt actually reached the provider (vs. a stale copy or a mutation).
func PromptHash(rendered string) string {
	sum := sha256.Sum256([]byte(rendered))
	return hex.EncodeToString(sum[:8]) // 16 hex chars is plenty for a fingerprint
}
