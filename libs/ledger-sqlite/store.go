// Package sqlite is the durable ledger event store (ADR-0034, ADR-0035).
//
// An adapter, so the constructs the domain forbids are legitimate here: I/O,
// `context`, mutable state, a driver. That asymmetry is the architecture
// working — a purity rule that fired in this package would be a rule nobody
// kept.
//
// A separate module rather than a package inside `libs/ledger`, because Go
// resolves dependencies per module: the driver would otherwise land in the
// `go.sum` of every consumer that imports `libs/ledger/domain`, including
// consumers that never touch storage (ADR-0013).
package sqlite

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"google.golang.org/protobuf/proto"

	ledgerv1 "github.com/FabioCaffarello/fdos/libs/contracts/gen/fdos/ledger/v1"
	ledgerwire "github.com/FabioCaffarello/fdos/libs/ledger-wire"
	"github.com/FabioCaffarello/fdos/libs/ledger/app"
	"github.com/FabioCaffarello/fdos/libs/ledger/domain"

	_ "modernc.org/sqlite" // the pure-Go driver ADR-0035 chose
)

// schema is applied on Open and is idempotent.
//
// One table and one index. The index covers the total order ADR-0009 defines —
// (effective_from, knowledge, sequence) within a stream — so an as-of read is a
// range scan rather than a full scan.
//
// The index is **derived state** and Constitution §1 governs it: it is not a
// second source of truth, and dropping it must change performance and nothing
// else. `TestIndexIsRebuildable` is what holds that.
//
// STRICT because a ledger that silently coerces a type is a ledger that
// silently changes a fact.
const schema = `
CREATE TABLE IF NOT EXISTS facts (
    stream          TEXT    NOT NULL,
    sequence        INTEGER NOT NULL,
    effective_from  TEXT    NOT NULL,
    knowledge       TEXT    NOT NULL,
    encoded         BLOB    NOT NULL,
    PRIMARY KEY (stream, sequence)
) STRICT;

CREATE INDEX IF NOT EXISTS facts_as_of ON facts (stream, effective_from, knowledge, sequence);
`

// pragmas are durability settings, not tuning.
//
// ADR-0035 recorded crash-safety as the gap its audit did not close, and named
// asserting these rather than assuming a default as the minimum. `journal_mode`
// and `synchronous` are what decide whether a committed fact survives losing
// power; `foreign_keys` and `busy_timeout` are hygiene.
var pragmas = []string{
	"PRAGMA journal_mode = WAL",
	"PRAGMA synchronous = FULL",
	"PRAGMA foreign_keys = ON",
	"PRAGMA busy_timeout = 5000",
}

// Store is a durable app.Store backed by SQLite.
type Store struct {
	db *sql.DB
}

// Open opens or creates the database at dsn and applies the schema.
//
// `dsn` is a file path. There is deliberately no in-memory convenience
// constructor: `libs/ledger/adapters/memory` is the in-memory store, and a
// second one here would be a second implementation of the same thing whose
// divergence nobody would notice.
func Open(dsn string) (*Store, error) {
	// `_txlock=immediate` makes every transaction take its write lock at BEGIN
	// rather than at the first write. See [Store.Append] for why that matters.
	db, err := sql.Open("sqlite", dsn+"?_txlock=immediate")
	if err != nil {
		return nil, fmt.Errorf("sqlite: open: %w", err)
	}

	// SQLite is single-writer. More connections do not buy concurrency here;
	// they buy `database is locked` under contention, which would surface as a
	// spurious error rather than as the serialisation it actually is.
	db.SetMaxOpenConns(1)

	for _, pragma := range pragmas {
		if _, err := db.Exec(pragma); err != nil {
			return nil, errors.Join(fmt.Errorf("sqlite: %s: %w", pragma, err), db.Close())
		}
	}
	if _, err := db.Exec(schema); err != nil {
		return nil, errors.Join(fmt.Errorf("sqlite: schema: %w", err), db.Close())
	}
	return &Store{db: db}, nil
}

// Close releases the database.
func (s *Store) Close() error { return s.db.Close() }

// Load rebuilds the stream from its facts, or returns app.ErrStreamNotFound.
//
// The stream is replayed through `domain.Stream.Append` rather than
// reconstructed field by field, so a decoded fact goes through the same
// constructor a new one does. A store that assembled a Stream directly could
// produce one the domain would refuse to build.
func (s *Store) Load(ctx context.Context, name string) (loaded domain.Stream, err error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT encoded FROM facts WHERE stream = ? ORDER BY sequence`, name)
	if err != nil {
		return domain.Stream{}, fmt.Errorf("sqlite: load %s: %w", name, err)
	}
	// Joined rather than blanked: a rows.Close failure can mean a truncated
	// read, and a truncated read of a ledger returned as success is the
	// quietest way to lose a fact.
	defer func() { err = errors.Join(err, rows.Close()) }()

	stream, err := domain.NewStream(name)
	if err != nil {
		return domain.Stream{}, fmt.Errorf("sqlite: %w", err)
	}

	found := false
	for rows.Next() {
		var encoded []byte
		if err := rows.Scan(&encoded); err != nil {
			return domain.Stream{}, fmt.Errorf("sqlite: scan %s: %w", name, err)
		}
		fact, err := decodeFact(encoded)
		if err != nil {
			return domain.Stream{}, fmt.Errorf("sqlite: %s: %w", name, err)
		}
		stream, _, err = stream.Append(fact.Envelope(), fact.Kind(), fact.Payload())
		if err != nil {
			return domain.Stream{}, fmt.Errorf("sqlite: replay %s: %w", name, err)
		}
		found = true
	}
	if err := rows.Err(); err != nil {
		return domain.Stream{}, fmt.Errorf("sqlite: load %s: %w", name, err)
	}
	if !found {
		return domain.Stream{}, fmt.Errorf("%w: %s", app.ErrStreamNotFound, name)
	}
	return stream, nil
}

// Append records one fact and returns the reference the store assigned.
//
// Everything happens inside one immediate transaction: reading the current
// length, checking the caller's expectation, checking knowledge-time
// monotonicity, and the insert. That is the whole point of moving the append
// here — the sequence is assigned where writes serialise, so two writers cannot
// compute the same Ref (ADR-0034).
//
// The transaction is IMMEDIATE, set on the connection in [Open]. A deferred
// transaction — the default — takes its write lock at the first write, which
// would leave the read that decides the sequence outside the lock: the same
// time-of-check gap this design exists to close, reintroduced one layer down.
func (s *Store) Append(
	ctx context.Context,
	name string,
	expect app.Expectation,
	envelope domain.Envelope,
	kind domain.Kind,
	payload domain.Payload,
) (ref domain.Ref, err error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return domain.Ref{}, fmt.Errorf("sqlite: begin: %w", err)
	}
	// sql.ErrTxDone is the expected outcome after a successful commit and is the
	// only rollback error that means nothing went wrong.
	defer func() {
		if rerr := tx.Rollback(); rerr != nil && !errors.Is(rerr, sql.ErrTxDone) {
			err = errors.Join(err, rerr)
		}
	}()

	var length int64
	var lastKnowledge sql.NullString
	err = tx.QueryRowContext(ctx,
		`SELECT COUNT(*), MAX(knowledge) FROM facts WHERE stream = ?`, name).
		Scan(&length, &lastKnowledge)
	if err != nil {
		return domain.Ref{}, fmt.Errorf("sqlite: read %s: %w", name, err)
	}

	if want, checked := expect.Length(); checked && int(length) != want {
		return domain.Ref{}, fmt.Errorf(
			"%w: %s held %d facts when read, holds %d now", app.ErrStaleRead, name, want, length)
	}

	knowledge := envelope.Coordinates().Knowledge()
	if lastKnowledge.Valid && knowledge.String() <= lastKnowledge.String {
		return domain.Ref{}, fmt.Errorf(
			"%w: %s last knew at %s, this append carries %s",
			app.ErrNonMonotonicKnowledge, name, lastKnowledge.String, knowledge)
	}

	// COUNT(*) cannot be negative; the guard states it rather than assuming it,
	// so the conversion below is checked rather than hoped.
	if length < 0 {
		return domain.Ref{}, fmt.Errorf("sqlite: %s reported %d facts", name, length)
	}
	ref = domain.Ref{Stream: name, Sequence: uint64(length) + 1} //nolint:gosec // guarded non-negative above
	fact, err := domain.NewFact(ref, envelope, kind, payload)
	if err != nil {
		return domain.Ref{}, fmt.Errorf("sqlite: %w", err)
	}
	encoded, err := encodeFact(fact)
	if err != nil {
		return domain.Ref{}, fmt.Errorf("sqlite: %w", err)
	}

	if _, err := tx.ExecContext(ctx,
		`INSERT INTO facts (stream, sequence, effective_from, knowledge, encoded) VALUES (?, ?, ?, ?, ?)`,
		name, ref.Sequence, envelope.Coordinates().Effective().From().String(), knowledge.String(), encoded,
	); err != nil {
		return domain.Ref{}, fmt.Errorf("sqlite: insert %s: %w", name, err)
	}
	if err := tx.Commit(); err != nil {
		return domain.Ref{}, fmt.Errorf("sqlite: commit %s: %w", name, err)
	}
	return ref, nil
}

// encodeFact renders a fact as the protobuf libs/ledger-wire already defines.
//
// Reusing the wire codec rather than inventing a storage encoding is ADR-0034's
// decision: two encodings of a fact would need a conformance suite proving they
// agree, which is a cost M7 paid once and should not pay twice.
func encodeFact(f domain.Fact) ([]byte, error) {
	wire, err := ledgerwire.EncodeFact(f)
	if err != nil {
		return nil, fmt.Errorf("encode: %w", err)
	}
	// Deterministic marshalling: the same fact must produce the same bytes, or
	// a byte-level comparison of two stores is meaningless (Constitution §9).
	encoded, err := proto.MarshalOptions{Deterministic: true}.Marshal(wire)
	if err != nil {
		return nil, fmt.Errorf("marshal: %w", err)
	}
	return encoded, nil
}

func decodeFact(encoded []byte) (domain.Fact, error) {
	var wire ledgerv1.Fact
	if err := proto.Unmarshal(encoded, &wire); err != nil {
		return domain.Fact{}, fmt.Errorf("unmarshal: %w", err)
	}
	fact, err := ledgerwire.DecodeFact(&wire)
	if err != nil {
		return domain.Fact{}, fmt.Errorf("decode: %w", err)
	}
	return fact, nil
}

// Compile-time proof that this satisfies the port. Without it, a signature drift
// would only surface wherever the store happened to be wired up.
var _ app.Store = (*Store)(nil)
