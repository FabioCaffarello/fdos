package app

import (
	"hash/fnv"
	"sync"
)

// shards is the size of the lock table.
//
// A constant rather than a knob. Tuning it trades memory for collisions between
// unrelated streams, and neither side of that trade is a correctness question —
// so a configuration option would be one more thing to get wrong in a
// composition root for no property gained (ADR-0036).
//
// 256 mutexes is a few kilobytes for the lifetime of the process.
const shards = 256

// streamLocks serialises writes to a stream, so that reading knowledge time and
// appending happen as one step.
//
// # Why this exists
//
// `Ledger` reads the clock, builds an envelope, and only then reaches the store,
// where writes serialise. A writer that read the clock earlier and arrived later
// is refused by the store's monotonicity check — correctly, and constantly.
// Measured on linux/amd64 before this existed: 32 concurrent admissions to one
// stream, 3 to 9 admitted, the rest refused with ErrNonMonotonicKnowledge. A
// clock handing every caller a distinct instant did not help, because the
// problem is the gap rather than the reading (ADR-0036).
//
// # Why the table is fixed rather than one lock per stream
//
// Stream names are producer-supplied and unbounded (ADR-0030). A map keyed by
// name is an allocation primitive handed to a stranger: one entry per fabricated
// name, from a caller that never has to succeed. Nothing bounds it, because
// nothing knows who may name a stream — that is D2, and it is open.
//
// A bound that depends on an open question is not a bound. This one is a fixed
// array, so there is no code path that grows it. The cost is that two unrelated
// streams occasionally share a lock and serialise; that is throughput, not
// correctness.
//
// # What it does not do
//
// It serialises writers **inside one process**. Two processes against one store
// reintroduce the gap exactly as ADR-0034 described, and the store's check
// remains the only thing between them and a broken ordering. Nothing here should
// be read as making a multi-writer deployment safe.
//
// It also does not make the wall clock finer. Monotonic means *strictly*
// greater, so two serialised appends that read the same instant are still
// refused — the ceiling per stream is the larger of one append latency and one
// clock tick. It does not bind on the platforms measured (nanoseconds on
// linux/amd64; microseconds on darwin/arm64, where the critical section is
// longer than a tick), and it would bind on a coarser clock or a faster store.
type streamLocks struct {
	mu [shards]sync.Mutex
}

// hold takes the lock covering `stream` and returns the release.
//
// Used as `defer l.writes.hold(name)()` — the lock is taken when the deferred
// expression is evaluated, and released when the use case returns.
func (s *streamLocks) hold(stream string) func() {
	m := &s.mu[shardFor(stream)]
	m.Lock()
	return m.Unlock
}

// shardFor maps a stream name onto the lock table.
//
// FNV-1a because it is in the standard library, it is deterministic across
// processes and builds, and nothing here depends on it being hard to collide
// with deliberately: a caller that forced two of its own streams onto one shard
// would slow itself down and nothing else.
func shardFor(stream string) uint32 {
	h := fnv.New32a()
	// Hash.Write never returns an error; the interface has one for io.Writer.
	_, _ = h.Write([]byte(stream))
	return h.Sum32() % shards
}
