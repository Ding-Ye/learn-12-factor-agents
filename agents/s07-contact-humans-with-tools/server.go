package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
)

type Server struct {
	Store    ThreadStore
	Provider Provider
	Registry Registry

	// BaseURL is included in the response_url field; tests can override.
	BaseURL string
}

type CreateRequest struct {
	Message string `json:"message"`
}

type ResponseRequest struct {
	Message string `json:"message"`
}

// ThreadView extends s06's view with `awaiting` + `response_url` so a
// client can detect and resume the loop without re-fetching state.
type ThreadView struct {
	ID          string  `json:"thread_id"`
	Thread      *Thread `json:"thread"`
	Awaiting    bool    `json:"awaiting,omitempty"`
	ResponseURL string  `json:"response_url,omitempty"`
}

func (s *Server) Handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("POST /thread", s.handleCreate)
	mux.HandleFunc("GET /thread/{id}", s.handleGet)
	mux.HandleFunc("POST /thread/{id}/response", s.handleResponse)
	return mux
}

func (s *Server) baseURL() string {
	if s.BaseURL != "" {
		return s.BaseURL
	}
	return ""
}

func (s *Server) view(id string, thread *Thread) ThreadView {
	v := ThreadView{ID: id, Thread: thread}
	if thread.AwaitingHuman() {
		v.Awaiting = true
		v.ResponseURL = fmt.Sprintf("%s/thread/%s/response", s.baseURL(), id)
	}
	return v
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
		s.Store.Update(id, thread)
		writeJSON(w, http.StatusInternalServerError, map[string]any{
			"error":     err.Error(),
			"thread_id": id,
			"thread":    thread,
		})
		return
	}
	s.Store.Update(id, final)
	writeJSON(w, http.StatusOK, s.view(id, final))
}

func (s *Server) handleGet(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	thread, ok := s.Store.Get(id)
	if !ok {
		http.Error(w, fmt.Sprintf("thread %q not found", id), http.StatusNotFound)
		return
	}
	writeJSON(w, http.StatusOK, s.view(id, thread))
}

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
			"error": err.Error(), "thread_id": id, "thread": thread,
		})
		return
	}
	s.Store.Update(id, final)
	writeJSON(w, http.StatusOK, s.view(id, final))
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}
