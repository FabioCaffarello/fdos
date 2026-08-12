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

	// writes is the region Serialise hands out, per stream. Distinct from `mu`,
	// which guards the map for the duration of a single method: a region spans a
	// caller's clock read and its append, and holding `mu` across those would
	// block every read of every other stream for no property gained.
	writes streamLocks
}

// NewStore creates an empty store.
func NewStore() *Store {
	return &Store{streams: make(map[string]domain.Stream)}
}

// Serialise runs fn holding the write region for `name` (ADR-0041).
//
// For an in-memory store, "every writer holding this store" is every writer
// that shares the process, because a second process cannot reach this map. That
// makes the guarantee complete here rather than partial — the same sentence
// that is a limitation for a durable adapter.
func (s *Store) Serialise(
	ctx context.Context,
	name string,
	fn func(context.Context, app.Store) error,
) error {
	defer s.writes.hold(name)()
	return fn(ctx, scoped{s})
}

// scoped is this store inside a region.
//
// It exists to refuse a nested Serialise. Load and Append are promoted from the
// embedded store and behave identically; only re-entry differs, and it differs
// because the alternative is deadlocking against a lock the caller already
// holds — a hang with no message rather than an error with one.
type scoped struct{ *Store }

// Serialise refuses: this store is already inside a region.
func (scoped) Serialise(context.Context, string, func(context.Context, app.Store) error) error {
	return app.ErrNestedSerialise
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
