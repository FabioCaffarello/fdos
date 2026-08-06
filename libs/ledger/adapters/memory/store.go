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

// Append records one fact against the authoritative stream and returns the ref
// the store assigned (ADR-0034).
//
// This is the whole point of the port change. The sequence comes from the length
// of the stream *held here*, under the lock, rather than from whatever length a
// caller happened to read — so two concurrent appends receive consecutive refs
// instead of the same one.
//
// The stream is created on first append: a stream is the facts in it, and there
// is nothing to declare in advance.
func (s *Store) Append(
	_ context.Context,
	name string,
	expect app.Expectation,
	envelope domain.Envelope,
	kind domain.Kind,
	payload domain.Payload,
) (domain.Ref, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	stream, ok := s.streams[name]
	if !ok {
		created, err := domain.NewStream(name)
		if err != nil {
			return domain.Ref{}, fmt.Errorf("memory: %w", err)
		}
		stream = created
	}

	if want, checked := expect.Length(); checked && stream.Len() != want {
		return domain.Ref{}, fmt.Errorf(
			"%w: %s held %d facts when read, holds %d now", app.ErrStaleRead, name, want, stream.Len())
	}

	// Knowledge time is monotonic per stream (ADR-0009). One writer satisfies
	// that by construction; two satisfy it only because of this.
	if stream.Len() > 0 {
		facts := stream.Facts()
		last := facts[len(facts)-1].Envelope().Coordinates().Knowledge()
		if incoming := envelope.Coordinates().Knowledge(); !incoming.After(last) {
			return domain.Ref{}, fmt.Errorf(
				"%w: %s last knew at %s, this append carries %s",
				app.ErrNonMonotonicKnowledge, name, last, incoming)
		}
	}

	extended, ref, err := stream.Append(envelope, kind, payload)
	if err != nil {
		return domain.Ref{}, fmt.Errorf("memory: %w", err)
	}
	s.streams[name] = extended
	return ref, nil
}
