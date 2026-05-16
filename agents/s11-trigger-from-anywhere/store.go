package main

import (
	"crypto/rand"
	"encoding/hex"
	"sync"
)

type ThreadStore interface {
	Create(t *Thread) string
	Get(id string) (*Thread, bool)
	Update(id string, t *Thread)
}

type MemoryStore struct {
	mu      sync.Mutex
	threads map[string]*Thread
}

func NewMemoryStore() *MemoryStore { return &MemoryStore{threads: make(map[string]*Thread)} }

func (s *MemoryStore) Create(t *Thread) string {
	id := newID()
	s.mu.Lock()
	defer s.mu.Unlock()
	s.threads[id] = t
	return id
}

func (s *MemoryStore) Get(id string) (*Thread, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	t, ok := s.threads[id]
	return t, ok
}

func (s *MemoryStore) Update(id string, t *Thread) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.threads[id] = t
}

func newID() string {
	b := make([]byte, 12)
	if _, err := rand.Read(b); err != nil {
		return "fallback-id-" + hex.EncodeToString([]byte{0})
	}
	return hex.EncodeToString(b)
}
