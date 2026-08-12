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
	"time"

	"google.golang.org/protobuf/proto"

	ledgerv1 "github.com/FabioCaffarello/fdos/libs/contracts/gen/fdos/ledger/v1"
	"github.com/FabioCaffarello/fdos/libs/kernel/temporal"
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
// silently changes a fact. Note what STRICT does *not* buy: it permits a
// lossless integer-to-text conversion, so it would not have caught the encoding
// change below arriving in a file whose columns were still TEXT.
//
// The temporal columns are **integer nanoseconds** since the Unix epoch, not
// rendered timestamps (ADR-0040). Byte order must equal chronological order,
// because the index above is what an as-of read range-scans, and a formatted
// timestamp does not give that: RFC 3339 §5.1 grants string-sortability only
// when every value carries the same number of fractional digits, and Go's
// RFC3339Nano strips trailing zeros. Measured, `…T00:00:00.000000001Z` compared
// as *earlier* than `…T00:00:00Z`, so this store refused appends the in-memory
// store accepted.
//
// The wire format was already right — `fdos.kernel.v1` carries a
// `google.protobuf.Timestamp`, an orderable integer pair. This restores one
// layer down the representation the contract had chosen, rather than inventing
// one.
const schema = `
CREATE TABLE IF NOT EXISTS facts (
    stream          TEXT    NOT NULL,
    sequence        INTEGER NOT NULL,
    effective_from  INTEGER NOT NULL,
    knowledge       INTEGER NOT NULL,
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
	if err := assertEncoding(db); err != nil {
		return nil, errors.Join(err, db.Close())
	}
	if _, err := db.Exec(schema); err != nil {
		return nil, errors.Join(fmt.Errorf("sqlite: schema: %w", err), db.Close())
	}
	return &Store{db: db}, nil
}

// Encoding versions recorded in `PRAGMA user_version` (ADR-0040).
const (
	// encodingRFC3339Text is the original format: temporal columns as rendered
	// RFC3339Nano strings, ordered lexicographically and therefore wrongly. No
	// store ever wrote this marker, because the pragma was never set — such a
	// file is identified by holding facts while reporting version zero.
	encodingRFC3339Text = 1

	// encodingOrderableInteger is the current format: temporal columns as
	// integer nanoseconds since the Unix epoch.
	encodingOrderableInteger = 2
)

// ErrEncodingVersion is returned when a database was written under an encoding
// this build cannot order correctly.
//
// Refusing rather than migrating is the decision, not a limitation (ADR-0040): a
// store whose facts this build cannot order is one it must not answer an as-of
// query from, and an as-of answer is the only thing the ledger is for.
var ErrEncodingVersion = errors.New("sqlite: database uses an older temporal encoding")

// assertEncoding reads the format marker and refuses what it cannot order.
//
// The awkward case is version **zero**, and it is why this reads the table as
// well as the pragma. A fresh file reports zero because nothing has set the
// pragma yet; a pre-ADR-0040 file reports zero for the same reason, having been
// written before the marker existed. The two are distinguished by whether any
// fact is present — an empty file has nothing to misorder, and a populated one
// holds text where this build expects integers.
//
// Without the marker the change would be unsafe in a way nothing reports.
// Measured: `CREATE TABLE IF NOT EXISTS` is a no-op against an existing file, so
// its columns stay TEXT; a `STRICT` TEXT column then *silently coerces* an
// integer to text; and the resulting mixed store orders `1782000000000000000`
// before `2026-07-01T00:00:00Z`. A store written before the fix would keep
// answering wrongly after it, forever, with no error.
func assertEncoding(db *sql.DB) error {
	var version int
	if err := db.QueryRow(`PRAGMA user_version`).Scan(&version); err != nil {
		return fmt.Errorf("sqlite: read user_version: %w", err)
	}

	switch version {
	case encodingOrderableInteger:
		return nil

	case 0:
		// `facts` may not exist yet, which is itself an empty store.
		var facts int
		err := db.QueryRow(
			`SELECT COUNT(*) FROM sqlite_master WHERE type = 'table' AND name = 'facts'`).Scan(&facts)
		if err != nil {
			return fmt.Errorf("sqlite: inspect schema: %w", err)
		}
		if facts > 0 {
			var rows int
			if err := db.QueryRow(`SELECT COUNT(*) FROM facts`).Scan(&rows); err != nil {
				return fmt.Errorf("sqlite: count facts: %w", err)
			}
			if rows > 0 {
				return fmt.Errorf(
					"%w: %d fact(s) written under encoding %d (RFC3339 text) and this build "+
						"expects %d (integer nanoseconds). Rebuild the table from the encoded "+
						"facts, which already carry both temporal coordinates, then set "+
						"PRAGMA user_version = %d",
					ErrEncodingVersion, rows, encodingRFC3339Text, encodingOrderableInteger,
					encodingOrderableInteger)
			}
		}
		// Empty, so there is nothing to misorder: claim it.
		if _, err := db.Exec(
			fmt.Sprintf(`PRAGMA user_version = %d`, encodingOrderableInteger)); err != nil {
			return fmt.Errorf("sqlite: set user_version: %w", err)
		}
		return nil

	default:
		return fmt.Errorf(
			"%w: user_version is %d and this build understands %d",
			ErrEncodingVersion, version, encodingOrderableInteger)
	}
}

// Close releases the database.
func (s *Store) Close() error { return s.db.Close() }

// ErrGap is returned when a stream's stored sequences are not 1..N contiguous.
//
// An append-only stream whose sequence is assigned by the store (ADR-0034) has
// no legitimate source of gaps: Append never reserves a number it might not use,
// which is what makes gaplessness free here and expensive in a database sequence.
// So a gap is not a condition to tolerate — it is evidence that rows were
// deleted or the file was altered out of band, and the only safe response is to
// refuse the stream and say which sequence is missing.
//
// Tolerating one is worse than it sounds. Replay assigns refs by position, so a
// single missing row silently re-points every later ref at different content: a
// FactCorrected naming s#3 would correct whatever landed at position 3 instead.
var ErrGap = errors.New("sqlite: stream has a sequence gap")

// Load rebuilds the stream from its facts, or returns app.ErrStreamNotFound.
//
// The stream is replayed through `domain.Stream.Append` rather than
// reconstructed field by field, so a decoded fact goes through the same
// constructor a new one does. A store that assembled a Stream directly could
// produce one the domain would refuse to build.
//
// Replay assigns each ref from the stream's current length, so it reproduces the
// stored sequences exactly when they are 1..N contiguous — and silently
// renumbers them when they are not. The stored sequence is therefore read and
// compared rather than discarded, which is what makes the replay's assumption
// checked instead of assumed. The sequence is stored twice, in the column and
// inside the encoded fact's own ref, and both are compared: they cannot
// disagree unless the row was written by something other than Append.
func (s *Store) Load(ctx context.Context, name string) (loaded domain.Stream, err error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT sequence, encoded FROM facts WHERE stream = ? ORDER BY sequence`, name)
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
		var stored int64
		var encoded []byte
		if err := rows.Scan(&stored, &encoded); err != nil {
			return domain.Stream{}, fmt.Errorf("sqlite: scan %s: %w", name, err)
		}
		fact, err := decodeFact(encoded)
		if err != nil {
			return domain.Stream{}, fmt.Errorf("sqlite: %s: %w", name, err)
		}

		// What replay is about to assign. Rows arrive ordered by sequence, so a
		// contiguous stream has stored == expected at every step.
		//
		// len() cannot be negative; the guard states it rather than assuming it, so
		// the widening below is checked rather than hoped — the same shape Append
		// uses for the same reason.
		replayed := stream.Len()
		if replayed < 0 {
			return domain.Stream{}, fmt.Errorf("sqlite: %s replayed %d facts", name, replayed)
		}
		expected := uint64(replayed) + 1              //nolint:gosec // guarded non-negative above
		if stored < 0 || uint64(stored) != expected { //nolint:gosec // guarded non-negative here
			return domain.Stream{}, fmt.Errorf(
				"%w: %s expected sequence %d, found %d — replay would renumber every later fact",
				ErrGap, name, expected, stored)
		}
		if got := fact.Ref().Sequence; got != expected {
			return domain.Stream{}, fmt.Errorf(
				"%w: %s row %d carries ref %s#%d — the column and the encoded fact disagree",
				ErrGap, name, expected, name, got)
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
	var highest sql.NullInt64
	var lastKnowledge sql.NullInt64
	err = tx.QueryRowContext(ctx,
		`SELECT COUNT(*), MAX(sequence), MAX(knowledge) FROM facts WHERE stream = ?`, name).
		Scan(&length, &highest, &lastKnowledge)
	if err != nil {
		return domain.Ref{}, fmt.Errorf("sqlite: read %s: %w", name, err)
	}

	// COUNT and MAX answer two different questions, and conflating them was the
	// defect: COUNT is how many facts survive, MAX is what position the last one
	// holds. They are equal exactly when the sequences are 1..N contiguous, so
	// comparing them is a whole-stream integrity check at no extra cost — no
	// second query, no row scan. Appending onto a gap would compound it.
	if highest.Valid && highest.Int64 != length {
		return domain.Ref{}, fmt.Errorf(
			"%w: %s holds %d facts with highest sequence %d",
			ErrGap, name, length, highest.Int64)
	}

	if want, checked := expect.Length(); checked && int(length) != want {
		return domain.Ref{}, fmt.Errorf(
			"%w: %s held %d facts when read, holds %d now", app.ErrStaleRead, name, want, length)
	}

	// Compared as instants, not as renderings. `MAX(knowledge)` over an integer
	// column is the chronological maximum; over the RFC3339 text this replaced it
	// was the lexicographic one, which picked the wrong row whenever any instant
	// carried a fraction (ADR-0040).
	knowledge := envelope.Coordinates().Knowledge()
	if lastKnowledge.Valid && nanos(knowledge) <= lastKnowledge.Int64 {
		return domain.Ref{}, fmt.Errorf(
			"%w: %s last knew at %s, this append carries %s",
			app.ErrNonMonotonicKnowledge, name, instantAt(lastKnowledge.Int64), knowledge)
	}

	// The next sequence comes from the highest one stored, never from a count of
	// surviving rows. Every mature event store assigns and stores the sequence
	// rather than recomputing it: COUNT(*)+1 collides with a live row the moment
	// any row is missing, which makes the stream permanently unappendable.
	//
	// The gap check above already proved COUNT and MAX agree, so this is the same
	// number today — but it is the correct number for the same reason a stored
	// sequence is the correct thing to load.
	if highest.Int64 < 0 {
		return domain.Ref{}, fmt.Errorf("sqlite: %s reported highest sequence %d", name, highest.Int64)
	}
	ref = domain.Ref{Stream: name, Sequence: uint64(highest.Int64) + 1} //nolint:gosec // guarded non-negative above
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
		name, ref.Sequence, nanos(envelope.Coordinates().Effective().From()), nanos(knowledge), encoded,
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
// nanos renders an instant as the store's temporal encoding: nanoseconds since
// the Unix epoch, which orders as bytes because it orders as a number.
//
// int64 nanoseconds spans the year 1678 to 2262, which is the same range Go's
// own time.Time.UnixNano covers and far wider than any instrument this ledger
// will hold. A fact outside it could not have been constructed, because
// temporal.At goes through time.Time.
func nanos(i temporal.Instant) int64 { return i.Time().UnixNano() }

// instantAt is nanos inverted, for error messages that name a stored instant
// rather than an integer nobody can read.
func instantAt(n int64) temporal.Instant {
	return temporal.MustAt(time.Unix(0, n).UTC())
}

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
