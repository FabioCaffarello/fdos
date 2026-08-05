// Package app orchestrates the ledger domain.
//
// This is the imperative shell around the pure core. It is where the things the
// domain refuses to touch live: the clock, `context.Context`, and the ports
// through which storage is reached (ADR-0013).
//
// Ports are declared **here**, not in `domain`. If they lived in the domain they
// would want `context.Context` in their signatures, and the ban on `context`
// there would collapse in the first week — which is exactly why RFC-0003 puts
// them in this layer.
package app

import (
	"context"
	"errors"

	"github.com/FabioCaffarello/fdos/libs/kernel/temporal"
	"github.com/FabioCaffarello/fdos/libs/ledger/domain"
)

// ErrStreamNotFound is returned by a Store when the named stream does not exist.
var ErrStreamNotFound = errors.New("app: stream not found")

// Clock supplies knowledge time.
//
// A port rather than a call to `time.Now()`, because a domain rule that reads
// the clock cannot be replayed (Constitution §2). Injecting it means the reading
// is an explicit input that becomes recorded provenance, and that a test can
// replay any history it likes.
type Clock interface {
	Now() temporal.Instant
}

// Store persists and retrieves fact streams.
//
// Append-only by construction: there is no Update and no Delete, because there
// is nothing the domain could ask an implementation to do with them
// (Constitution §4). An implementation offering more would be offering
// something no caller can reach.
type Store interface {
	// Load returns the stream, or ErrStreamNotFound.
	Load(ctx context.Context, name string) (domain.Stream, error)

	// Save records a stream that has grown by appending. An implementation
	// must reject a stream shorter than the one it holds: that would be a
	// rewrite of history arriving as a save.
	Save(ctx context.Context, stream domain.Stream) error
}
