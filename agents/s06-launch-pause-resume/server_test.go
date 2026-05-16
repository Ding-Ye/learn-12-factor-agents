package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
)

func newTestServer() *Server {
	return &Server{
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
}

func TestPostThread_ReturnsIDAndCompletedThread(t *testing.T) {
	srv := newTestServer()
	w := httptest.NewRecorder()
	body := bytes.NewBufferString(`{"message":"add 5 and 3, then multiply by 2"}`)
	r := httptest.NewRequest(http.MethodPost, "/thread", body)
	srv.Handler().ServeHTTP(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body=%s", w.Code, w.Body.String())
	}
	var got ThreadView
	if err := json.Unmarshal(w.Body.Bytes(), &got); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if got.ID == "" {
		t.Errorf("thread_id missing")
	}
	if len(got.Thread.Events) != 6 {
		t.Errorf("event count = %d, want 6", len(got.Thread.Events))
	}
}

func TestGetThread_RoundTripsState(t *testing.T) {
	srv := newTestServer()
	mux := srv.Handler()

	post := httptest.NewRequest(http.MethodPost, "/thread", bytes.NewBufferString(`{"message":"go"}`))
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, post)
	var view ThreadView
	_ = json.Unmarshal(w.Body.Bytes(), &view)

	get := httptest.NewRequest(http.MethodGet, "/thread/"+view.ID, nil)
	gw := httptest.NewRecorder()
	mux.ServeHTTP(gw, get)
	if gw.Code != http.StatusOK {
		t.Fatalf("GET status = %d, want 200; body=%s", gw.Code, gw.Body.String())
	}
	var got ThreadView
	_ = json.Unmarshal(gw.Body.Bytes(), &got)
	if len(got.Thread.Events) != len(view.Thread.Events) {
		t.Errorf("round-trip lost events: post=%d get=%d", len(view.Thread.Events), len(got.Thread.Events))
	}
}

func TestGetThread_NotFound(t *testing.T) {
	srv := newTestServer()
	r := httptest.NewRequest(http.MethodGet, "/thread/unknown-id", nil)
	w := httptest.NewRecorder()
	srv.Handler().ServeHTTP(w, r)
	if w.Code != http.StatusNotFound {
		t.Errorf("status = %d, want 404", w.Code)
	}
}

func TestPostResponse_AppendsHumanResponseAndResumes(t *testing.T) {
	// Provider that emits done_for_now first (so the initial POST
	// completes immediately) then add(1,2) → done on resume.
	srv := &Server{
		Store: NewMemoryStore(),
		Provider: &ScriptedSequenceProvider{
			Steps: []NextStep{
				doneStep("Initial done."),
				mathStep(IntentAdd, 1, 2),
				doneStep("After resume done."),
			},
		},
		Registry: DefaultRegistry(),
	}
	mux := srv.Handler()

	// initial POST → 1 turn done
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, httptest.NewRequest(http.MethodPost, "/thread", bytes.NewBufferString(`{"message":"hi"}`)))
	var view ThreadView
	_ = json.Unmarshal(w.Body.Bytes(), &view)

	// POST /thread/{id}/response → appends human_response + runs more
	w2 := httptest.NewRecorder()
	mux.ServeHTTP(w2, httptest.NewRequest(http.MethodPost, "/thread/"+view.ID+"/response",
		bytes.NewBufferString(`{"message":"please add 1 and 2"}`)))
	if w2.Code != http.StatusOK {
		t.Fatalf("response status = %d, body=%s", w2.Code, w2.Body.String())
	}
	var resumed ThreadView
	_ = json.Unmarshal(w2.Body.Bytes(), &resumed)

	if !containsHumanResponse(resumed.Thread) {
		t.Errorf("resumed thread missing human_response event:\n%+v", resumed.Thread.Events)
	}
}

func containsHumanResponse(t *Thread) bool {
	for _, e := range t.Events {
		if e.Type == EventTypeHumanResponse {
			return true
		}
	}
	return false
}

func TestPostThread_BadBody(t *testing.T) {
	srv := newTestServer()
	r := httptest.NewRequest(http.MethodPost, "/thread", strings.NewReader(`{not json`))
	w := httptest.NewRecorder()
	srv.Handler().ServeHTTP(w, r)
	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want 400", w.Code)
	}
}

func TestServer_ConcurrentCreatesDontRace(t *testing.T) {
	// Each request must get its own provider — ScriptedSequenceProvider
	// has internal state and goroutines would race on its .calls field.
	store := NewMemoryStore()
	registry := DefaultRegistry()

	var wg sync.WaitGroup
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			srv := &Server{
				Store: store,
				Provider: &ScriptedSequenceProvider{
					Steps: []NextStep{doneStep("done")},
				},
				Registry: registry,
			}
			w := httptest.NewRecorder()
			srv.Handler().ServeHTTP(w, httptest.NewRequest(http.MethodPost, "/thread", bytes.NewBufferString(`{"message":"x"}`)))
		}()
	}
	wg.Wait()
}
