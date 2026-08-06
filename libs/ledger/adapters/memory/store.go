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

// Save records a stream, refusing any write that does not extend the one held.
//
// The ledger is append-only (Constitution §4). A **shorter** stream arriving as
// a save is a rewrite of history wearing the costume of a write. An
// **equal-length** one is worse, and is the defect the M10 gate measured: two
// callers load the same stream, each append a different fact, each save at the
// same length, and the second silently overwrote the first. Both saves returned
// nil and nothing recorded that a fact had ever existed.
//
// Both appends are also assigned the *same* Ref, so the losing caller holds a
// reference that now addresses a different fact — including a Ref already
// recorded as a derivation input. A store may refuse a write; it may not accept
// one and drop another.
//
// # What this check is, and what it is not
//
// It is a length precondition, and it is deliberately the weakest thing that
// closes the measured defect. It catches the case where two callers append from
// the same base, which is what the load-append-save use cases actually do.
//
// It does **not** catch a caller that appends two facts while another appends
// one: the longer write is accepted and the shorter one's fact is still lost.
// Closing that needs a precondition the *caller* supplies — an expected length
// or an expected ref — which is a change to the `app.Store` port, and the shape
// of that port is the open question the M10 RFC exists to answer. Fixing it
// here would pre-empt that decision from an adapter.
//
// So: the silent loss is gone, the residual is named, and the port stays the
// RFC's to decide.
func (s *Store) Save(_ context.Context, stream domain.Stream) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	existing, ok := s.streams[stream.Name()]
	if ok && stream.Len() < existing.Len() {
		return fmt.Errorf(
			"memory: refusing to shorten stream %s from %d to %d facts; history is never rewritten",
			stream.Name(), existing.Len(), stream.Len(),
		)
	}
	if ok && stream.Len() == existing.Len() {
		return fmt.Errorf(
			"memory: refusing to overwrite stream %s at %d facts; a write that does not extend the stream "+
				"would drop a fact appended concurrently, and both would carry the same ref",
			stream.Name(), existing.Len(),
		)
	}
	s.streams[stream.Name()] = stream
	return nil
}
