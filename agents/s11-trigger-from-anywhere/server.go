package main

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/Ding-Ye/learn-12-factor-agents/s11-trigger-from-anywhere/triggers"
)

type Server struct {
	Store    ThreadStore
	Provider Provider
	Triggers map[string]triggers.Trigger
}

func (s *Server) Handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("POST /triggers/{name}", s.handleTrigger)
	mux.HandleFunc("GET /thread/{id}", s.handleGet)
	return mux
}

func (s *Server) handleGet(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	thread, ok := s.Store.Get(id)
	if !ok {
		http.Error(w, fmt.Sprintf("thread %q not found", id), http.StatusNotFound)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"thread_id": id, "thread": thread})
}

func (s *Server) handleTrigger(w http.ResponseWriter, r *http.Request) {
	name := r.PathValue("name")
	trigger, ok := s.Triggers[name]
	if !ok {
		http.Error(w, fmt.Sprintf("unknown trigger %q", name), http.StatusNotFound)
		return
	}

	outcome, err := trigger.Trigger(r)
	if err != nil {
		http.Error(w, "trigger parse: "+err.Error(), http.StatusBadRequest)
		return
	}

	switch {
	case outcome.IsFresh():
		thread := NewThread(
			NewTriggerEvent(trigger.Source(), outcome.Raw),
			NewUserInputEvent(outcome.FreshUserInput),
		)
		id := s.Store.Create(thread)
		final, err := RunAgent(r.Context(), thread, s.Provider)
		if err != nil {
			s.Store.Update(id, thread)
			writeJSON(w, http.StatusInternalServerError, map[string]any{"error": err.Error(), "thread_id": id, "thread": thread})
			return
		}
		s.Store.Update(id, final)
		writeJSON(w, http.StatusOK, map[string]any{"thread_id": id, "thread": final})

	case outcome.IsResume():
		thread, ok := s.Store.Get(outcome.ResumeThreadID)
		if !ok {
			http.Error(w, fmt.Sprintf("resume target %q not found", outcome.ResumeThreadID), http.StatusNotFound)
			return
		}
		thread.Append(NewHumanResponseEvent(outcome.HumanResponse))
		final, err := RunAgent(r.Context(), thread, s.Provider)
		if err != nil {
			s.Store.Update(outcome.ResumeThreadID, thread)
			writeJSON(w, http.StatusInternalServerError, map[string]any{"error": err.Error(), "thread_id": outcome.ResumeThreadID, "thread": thread})
			return
		}
		s.Store.Update(outcome.ResumeThreadID, final)
		writeJSON(w, http.StatusOK, map[string]any{"thread_id": outcome.ResumeThreadID, "thread": final})

	default:
		http.Error(w, "trigger produced neither fresh nor resume outcome", http.StatusInternalServerError)
	}
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}
