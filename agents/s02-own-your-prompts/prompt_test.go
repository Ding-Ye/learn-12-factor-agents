package main

import (
	"context"
	"strings"
	"testing"
)

func TestRenderPrompt_HasRoleMarkers(t *testing.T) {
	out, err := RenderPrompt(PromptInput{UserInput: "add 5 and 3"})
	if err != nil {
		t.Fatalf("RenderPrompt: %v", err)
	}
	if !strings.Contains(out, "SYSTEM:") {
		t.Errorf("rendered prompt missing SYSTEM: marker")
	}
	if !strings.Contains(out, "USER:") {
		t.Errorf("rendered prompt missing USER: marker")
	}
	if !strings.Contains(out, "add 5 and 3") {
		t.Errorf("rendered prompt missing user input")
	}
}

func TestRenderPrompt_SystemBeforeUser(t *testing.T) {
	out, err := RenderPrompt(PromptInput{UserInput: "x"})
	if err != nil {
		t.Fatalf("RenderPrompt: %v", err)
	}
	sysIdx := strings.Index(out, "SYSTEM:")
	userIdx := strings.Index(out, "USER:")
	if sysIdx == -1 || userIdx == -1 || sysIdx > userIdx {
		t.Errorf("expected SYSTEM: before USER:, got sysIdx=%d userIdx=%d", sysIdx, userIdx)
	}
}

func TestPromptHash_Stable(t *testing.T) {
	a, _ := RenderPrompt(PromptInput{UserInput: "hello"})
	b, _ := RenderPrompt(PromptInput{UserInput: "hello"})
	if PromptHash(a) != PromptHash(b) {
		t.Errorf("identical inputs produced different hashes: %s vs %s", PromptHash(a), PromptHash(b))
	}
}

func TestPromptHash_DifferentInputs(t *testing.T) {
	a, _ := RenderPrompt(PromptInput{UserInput: "hello"})
	b, _ := RenderPrompt(PromptInput{UserInput: "different"})
	if PromptHash(a) == PromptHash(b) {
		t.Errorf("different inputs produced the same hash — fingerprint is useless")
	}
}

func TestRecordingProvider_PassesThroughRenderedPrompt(t *testing.T) {
	rendered, _ := RenderPrompt(PromptInput{UserInput: "ping"})
	p := &RecordingProvider{}
	_, err := p.DetermineNextStep(context.Background(), rendered)
	if err != nil {
		t.Fatalf("DetermineNextStep: %v", err)
	}
	if p.LastSeen != rendered {
		t.Errorf("provider didn't see the rendered prompt verbatim:\n got: %q\nwant: %q", p.LastSeen, rendered)
	}
}

func TestRecordingProvider_EchoesPromptHashInMessage(t *testing.T) {
	rendered, _ := RenderPrompt(PromptInput{UserInput: "ping"})
	p := &RecordingProvider{}
	step, err := p.DetermineNextStep(context.Background(), rendered)
	if err != nil {
		t.Fatalf("DetermineNextStep: %v", err)
	}
	out, err := renderNextStep(step)
	if err != nil {
		t.Fatalf("renderNextStep: %v", err)
	}
	if !strings.Contains(out, PromptHash(rendered)) {
		t.Errorf("output missing prompt hash %q:\n%s", PromptHash(rendered), out)
	}
}
