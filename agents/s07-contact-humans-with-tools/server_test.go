package main

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestThread_AwaitingHuman_DetectsApproval(t *testing.T) {
	thread := NewThread(NewUserInputEvent("send money"))
	thread.Append(NewToolCallEvent(approvalStep("ok?", "high")))
	if !thread.AwaitingHuman() {
		t.Errorf("AwaitingHuman should be true after appending an approval tool_call")
	}
}

func TestThread_AwaitingHuman_FalseAfterDone(t *testing.T) {
	thread := NewThread(NewUserInputEvent("hi"))
	thread.Append(NewToolCallEvent(doneStep("bye")))
	if thread.AwaitingHuman() {
		t.Errorf("AwaitingHuman should be false after done_for_now")
	}
}

func TestRunAgent_BreaksOnHumanIntent(t *testing.T) {
	thread := NewThread(NewUserInputEvent("send money"))
	provider := &ScriptedSequenceProvider{
		Steps: []NextStep{
			approvalStep("Approve?", "high"),
			// these should never run
			mathStep(IntentAdd, 1, 1),
			doneStep("unreachable"),
		},
	}
	final, err := RunAgent(context.Background(), thread, provider, DefaultRegistry())
	if err != nil {
		t.Fatalf("RunAgent: %v", err)
	}
	if !final.AwaitingHuman() {
		t.Errorf("final thread should be AwaitingHuman")
	}
	if provider.calls != 1 {
		t.Errorf("provider should be called exactly once, got %d", provider.calls)
	}
}

func TestServer_RoundTrip_AwaitingHuman_Resume(t *testing.T) {
	srv := &Server{
		Store: NewMemoryStore(),
		Provider: &ScriptedSequenceProvider{
			Steps: []NextStep{
				clarifyStep("What's a, what's b?"),
				mathStep(IntentAdd, 2, 3),
				doneStep("5"),
			},
		},
		Registry: DefaultRegistry(),
		BaseURL:  "http://test.example",
	}
	mux := srv.Handler()

	// POST /thread → returns Awaiting=true + response_url
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, httptest.NewRequest(http.MethodPost, "/thread",
		bytes.NewBufferString(`{"message":"add them"}`)))
	if w.Code != http.StatusOK {
		t.Fatalf("POST status = %d, body=%s", w.Code, w.Body.String())
	}
	var view ThreadView
	_ = json.Unmarshal(w.Body.Bytes(), &view)
	if !view.Awaiting {
		t.Errorf("expected Awaiting=true; got %+v", view)
	}
	if view.ResponseURL == "" {
		t.Errorf("expected non-empty ResponseURL")
	}

	// POST /thread/{id}/response → loop resumes, completes
	w2 := httptest.NewRecorder()
	mux.ServeHTTP(w2, httptest.NewRequest(http.MethodPost, "/thread/"+view.ID+"/response",
		bytes.NewBufferString(`{"message":"a=2 b=3"}`)))
	if w2.Code != http.StatusOK {
		t.Fatalf("resume status = %d, body=%s", w2.Code, w2.Body.String())
	}
	var resumed ThreadView
	_ = json.Unmarshal(w2.Body.Bytes(), &resumed)
	if resumed.Awaiting {
		t.Errorf("after resume, Awaiting should be false")
	}
}

func TestRegistry_AllHumanToolsReturnErrHumanContact(t *testing.T) {
	for _, intent := range []string{IntentRequestApproval, IntentRequestMoreInformation} {
		tool, err := DefaultRegistry().Lookup(intent)
		if err != nil {
			t.Fatalf("Lookup %q: %v", intent, err)
		}
		_, err = tool.Execute(context.Background(), []byte(`{}`))
		if err != ErrHumanContact {
			t.Errorf("intent %q: Execute err = %v, want ErrHumanContact", intent, err)
		}
	}
}
