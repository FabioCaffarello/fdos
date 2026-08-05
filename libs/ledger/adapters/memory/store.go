// Package memory is an in-memory Store, for tests and for the M6 vertical
// slice.
//
// An adapter, so the constructs the domain forbids are legitimate here:
// `sync`, `context`, mutable state. That asymmetry is the architecture working
// — a purity rule that fired in this package would be a rule nobody kept.
//
// It is deliberately not a general-purpose store. There is no query language,
// no index and no persistence, because the slice exists to prove the boundaries
// hold, not to be a database.
package memory

import (
	"context"
	"fmt"
	"sync"

	"github.com/FabioCaffarello/fdos/libs/ledger/app"
	"github.com/FabioCaffarello/fdos/libs/ledger/domain"
)

// Store keeps streams in memory.
type Store struct {
	mu      sync.RWMutex
	streams map[string]domain.Stream
}

// NewStore creates an empty store.
func NewStore() *Store {
	return &Store{streams: make(map[string]domain.Stream)}
}

// Load returns a stream, or app.ErrStreamNotFound.
func (s *Store) Load(_ context.Context, name string) (domain.Stream, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	stream, ok := s.streams[name]
	if !ok {
		return domain.Stream{}, fmt.Errorf("%w: %s", app.ErrStreamNotFound, name)
	}
	return stream, nil
}

// Save records a stream.
//
// Rejects a stream shorter than the one held. The ledger is append-only
// (Constitution §4), and a shorter stream arriving as a save is a rewrite of
// history wearing the costume of a write — the one thing an append-only store
// must refuse rather than accept quietly.
func (s *Store) Save(_ context.Context, stream domain.Stream) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if existing, ok := s.streams[stream.Name()]; ok && stream.Len() < existing.Len() {
		return fmt.Errorf(
			"memory: refusing to shorten stream %s from %d to %d facts; history is never rewritten",
			stream.Name(), existing.Len(), stream.Len(),
		)
	}
	s.streams[stream.Name()] = stream
	return nil
}
