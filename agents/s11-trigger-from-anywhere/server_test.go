package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/Ding-Ye/learn-12-factor-agents/s11-trigger-from-anywhere/triggers"
)

func newTestServer() *Server {
	return &Server{
		Store: NewMemoryStore(),
		Provider: &ScriptedSequenceProvider{
			Steps: []NextStep{
				mathStep(2, 3),
				doneStep("done"),
			},
		},
		Triggers: map[string]triggers.Trigger{
			"slack":   triggers.SlackTrigger{},
			"webhook": triggers.WebhookTrigger{},
		},
	}
}

func TestSlackTrigger_SpawnsFreshThread(t *testing.T) {
	srv := newTestServer()
	body := strings.NewReader(`{"event":{"type":"message","text":"add 2 and 3","channel":"C1"}}`)
	w := httptest.NewRecorder()
	srv.Handler().ServeHTTP(w, httptest.NewRequest(http.MethodPost, "/triggers/slack", body))
	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", w.Code, w.Body.String())
	}
	var got map[string]any
	_ = json.Unmarshal(w.Body.Bytes(), &got)
	if got["thread_id"] == nil || got["thread_id"] == "" {
		t.Errorf("missing thread_id: %+v", got)
	}
}

func TestWebhookTrigger_ResumesExistingThread(t *testing.T) {
	srv := newTestServer()
	// Seed an existing thread.
	thread := NewThread(NewUserInputEvent("send money"))
	id := srv.Store.Create(thread)

	body := bytes.NewBufferString(`{"event":{"spec":{"state":{"thread_id":"` + id + `"}},"status":{"response":"approved"}}}`)
	w := httptest.NewRecorder()
	srv.Handler().ServeHTTP(w, httptest.NewRequest(http.MethodPost, "/triggers/webhook", body))
	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", w.Code, w.Body.String())
	}
	// Confirm human_response present in updated thread.
	got, _ := srv.Store.Get(id)
	found := false
	for _, e := range got.Events {
		if e.Type == EventTypeHumanResponse {
			found = true
		}
	}
	if !found {
		t.Errorf("resumed thread missing human_response")
	}
}

func TestUnknownTrigger_Returns404(t *testing.T) {
	srv := newTestServer()
	body := strings.NewReader(`{}`)
	w := httptest.NewRecorder()
	srv.Handler().ServeHTTP(w, httptest.NewRequest(http.MethodPost, "/triggers/unknown", body))
	if w.Code != http.StatusNotFound {
		t.Errorf("status = %d, want 404", w.Code)
	}
}

func TestSlackTrigger_BadShape(t *testing.T) {
	srv := newTestServer()
	body := strings.NewReader(`{"event":{"type":"reaction"}}`)
	w := httptest.NewRecorder()
	srv.Handler().ServeHTTP(w, httptest.NewRequest(http.MethodPost, "/triggers/slack", body))
	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want 400", w.Code)
	}
}

func TestWebhookTrigger_ResumeMissingID(t *testing.T) {
	srv := newTestServer()
	body := strings.NewReader(`{"event":{"spec":{"state":{"thread_id":"nope"}},"status":{"response":"ok"}}}`)
	w := httptest.NewRecorder()
	srv.Handler().ServeHTTP(w, httptest.NewRequest(http.MethodPost, "/triggers/webhook", body))
	if w.Code != http.StatusNotFound {
		t.Errorf("status = %d, want 404", w.Code)
	}
}
