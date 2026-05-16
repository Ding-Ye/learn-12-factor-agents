package main

import (
	"crypto/rand"
	"encoding/hex"
	"sync"
)

// ThreadStore is the persistent layer the HTTP handlers talk to. The
// in-memory implementation here is the only one s06 ships; the s12 work
// would be where you'd plug in sqlite or redis.
//
// Methods are all goroutine-safe — the HTTP server can serve concurrent
// requests for different threads without external locking.
type ThreadStore interface {
	Create(t *Thread) string
	Get(id string) (*Thread, bool)
	Update(id string, t *Thread)
}

type MemoryStore struct {
	mu      sync.Mutex
	threads map[string]*Thread
}

func NewMemoryStore() *MemoryStore {
	return &MemoryStore{threads: make(map[string]*Thread)}
}

// Create generates a 12-byte random hex id (24 chars). 12 bytes is
// plenty of entropy for an in-process map — we don't need full UUIDs.
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
		// crypto/rand should never fail. If it does, callers see a
		// predictable id and can panic; in practice this branch is
		// unreachable on Linux/macOS.
		return "fallback-id-" + hex.EncodeToString([]byte{0})
	}
	return hex.EncodeToString(b)
}
