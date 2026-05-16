package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
)

// Server bundles the three pieces s06 needs: thread storage, an LLM
// provider, and a tool registry. We pass them in so tests can construct
// the server with deterministic stubs.
type Server struct {
	Store    ThreadStore
	Provider Provider
	Registry Registry
}

// CreateRequest is the body shape of POST /thread.
type CreateRequest struct {
	Message string `json:"message"`
}

// CreateResponse / ThreadView is what the server returns.
type ThreadView struct {
	ID     string  `json:"thread_id"`
	Thread *Thread `json:"thread"`
}

// ResponseRequest is the body of POST /thread/{id}/response.
type ResponseRequest struct {
	Message string `json:"message"`
}

// Handler returns an http.Handler that routes the three endpoints.
// We use plain net/http to keep the dependency list to stdlib.
func (s *Server) Handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("POST /thread", s.handleCreate)
	mux.HandleFunc("GET /thread/{id}", s.handleGet)
	mux.HandleFunc("POST /thread/{id}/response", s.handleResponse)
	return mux
}

func (s *Server) handleCreate(w http.ResponseWriter, r *http.Request) {
	var req CreateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid body: "+err.Error(), http.StatusBadRequest)
		return
	}
	if strings.TrimSpace(req.Message) == "" {
		http.Error(w, "message is required", http.StatusBadRequest)
		return
	}

	thread := NewThread(NewUserInputEvent(req.Message))
	id := s.Store.Create(thread)

	final, err := RunAgent(r.Context(), thread, s.Provider, s.Registry)
	if err != nil {
		// Even on agent error we persist what we have — the caller can
		// inspect the partial thread to debug.
		s.Store.Update(id, thread)
		writeJSON(w, http.StatusInternalServerError, map[string]any{
			"error":     err.Error(),
			"thread_id": id,
			"thread":    thread,
		})
		return
	}
	s.Store.Update(id, final)
	writeJSON(w, http.StatusOK, ThreadView{ID: id, Thread: final})
}

func (s *Server) handleGet(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	thread, ok := s.Store.Get(id)
	if !ok {
		http.Error(w, fmt.Sprintf("thread %q not found", id), http.StatusNotFound)
		return
	}
	writeJSON(w, http.StatusOK, ThreadView{ID: id, Thread: thread})
}

// handleResponse appends a human_response event and re-runs the agent.
// s07 will make use of this when human-contact tools break the loop.
func (s *Server) handleResponse(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	thread, ok := s.Store.Get(id)
	if !ok {
		http.Error(w, fmt.Sprintf("thread %q not found", id), http.StatusNotFound)
		return
	}
	var req ResponseRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid body: "+err.Error(), http.StatusBadRequest)
		return
	}
	thread.Append(NewHumanResponseEvent(req.Message))

	final, err := RunAgent(r.Context(), thread, s.Provider, s.Registry)
	if err != nil {
		s.Store.Update(id, thread)
		writeJSON(w, http.StatusInternalServerError, map[string]any{
			"error":     err.Error(),
			"thread_id": id,
			"thread":    thread,
		})
		return
	}
	s.Store.Update(id, final)
	writeJSON(w, http.StatusOK, ThreadView{ID: id, Thread: final})
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

