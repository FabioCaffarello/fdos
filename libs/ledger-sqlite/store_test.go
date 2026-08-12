package sqlite_test

import (
	"context"
	"database/sql"
	"errors"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/FabioCaffarello/fdos/libs/kernel/identity"
	"github.com/FabioCaffarello/fdos/libs/kernel/money"
	"github.com/FabioCaffarello/fdos/libs/kernel/provenance"
	"github.com/FabioCaffarello/fdos/libs/kernel/temporal"
	sqlitestore "github.com/FabioCaffarello/fdos/libs/ledger-sqlite"
	"github.com/FabioCaffarello/fdos/libs/ledger/app"
	"github.com/FabioCaffarello/fdos/libs/ledger/domain"
	"github.com/FabioCaffarello/fdos/libs/ledger/storetest"
)

// The durable store is held to exactly the definition of `app.Store` the
// in-memory one is (ADR-0034). One suite, two implementations — which is why
// the suite lives in the context module rather than in either adapter.
func TestConformance(t *testing.T) {
	storetest.Run(t, func(t *testing.T) app.Store {
		return open(t, filepath.Join(t.TempDir(), "ledger.db"))
	})
}

// Case 11, and the point of the whole module: a fact outlives the process.
//
// This is the case the in-memory store cannot satisfy and therefore the one the
// shared suite cannot contain. Everything before M10 was true only until the
// program ended.
func TestFactsSurviveClosingTheDatabase(t *testing.T) {
	ctx := context.Background()
	path := filepath.Join(t.TempDir(), "ledger.db")

	first := open(t, path)
	want := []string{"PETR4", "VALE3", "ITUB4"}
	refs := make([]domain.Ref, 0, len(want))
	for i, ticker := range want {
		ref, err := first.Append(ctx, "acct-1", app.Any(), envelopeAt(t, i+1), domain.KindObservation, claim(t, ticker))
		if err != nil {
			t.Fatalf("append %s: %v", ticker, err)
		}
		refs = append(refs, ref)
	}
	if err := first.Close(); err != nil {
		t.Fatalf("close: %v", err)
	}

	// A different process, as far as this store is concerned.
	second := open(t, path)
	stream, err := second.Load(ctx, "acct-1")
	if err != nil {
		t.Fatalf("load after reopen: %v", err)
	}
	if stream.Len() != len(want) {
		t.Fatalf("after reopen the stream holds %d facts, want %d", stream.Len(), len(want))
	}
	for i, f := range stream.Facts() {
		if f.Ref() != refs[i] {
			t.Errorf("fact %d came back as %s, want %s", i, f.Ref(), refs[i])
		}
		held, ok := f.Payload().(domain.HoldingClaimed)
		if !ok || held.Instrument.Value() != want[i] {
			t.Errorf("fact %d came back as %v, want %s", i, f.Payload(), want[i])
		}
		if f.Envelope().Provenance().IsZero() {
			t.Errorf("fact %d came back without provenance", i)
		}
	}
}

// Case 12. The index is derived state, and Constitution §1 says derived state
// is not a second source of truth.
//
// Dropping it must change performance and nothing else. An index that cannot be
// rebuilt from the facts is a second copy of the ledger that will eventually
// disagree with the first, with nothing to say which is right.
func TestTheIndexIsDerivedAndRebuildable(t *testing.T) {
	ctx := context.Background()
	path := filepath.Join(t.TempDir(), "ledger.db")

	store := open(t, path)
	for i, ticker := range []string{"PETR4", "VALE3", "ITUB4"} {
		if _, err := store.Append(
			ctx, "acct-1", app.Any(), envelopeAt(t, i+1), domain.KindObservation, claim(t, ticker),
		); err != nil {
			t.Fatal(err)
		}
	}
	before, err := store.Load(ctx, "acct-1")
	if err != nil {
		t.Fatal(err)
	}
	if closeErr := store.Close(); closeErr != nil {
		t.Fatal(closeErr)
	}

	raw := openRaw(t, path)
	if _, dropErr := raw.Exec(`DROP INDEX facts_as_of`); dropErr != nil {
		t.Fatalf("drop index: %v", dropErr)
	}
	if closeErr := raw.Close(); closeErr != nil {
		t.Fatal(closeErr)
	}

	// Open recreates the index from the schema; the facts were never touched.
	rebuilt := open(t, path)
	after, err := rebuilt.Load(ctx, "acct-1")
	if err != nil {
		t.Fatalf("load after dropping the index: %v", err)
	}
	if after.Len() != before.Len() {
		t.Fatalf("rebuilding the index changed the answer: %d facts, was %d", after.Len(), before.Len())
	}
	for i := range before.Facts() {
		if before.Facts()[i].Ref() != after.Facts()[i].Ref() {
			t.Errorf("fact %d changed identity across an index rebuild", i)
		}
	}
}

// Case 13, the storage analogue of `make repro-check` (ADR-0034).
//
// Replaying a stream into a fresh database must produce not merely equal
// answers but **byte-identical encodings** — that is what makes a derivation's
// content address stable, and a content address that moved would silently
// invalidate every trace pointing at it.
func TestReplayIntoAFreshDatabaseIsByteIdentical(t *testing.T) {
	ctx := context.Background()
	dir := t.TempDir()

	write := func(path string) [][]byte {
		store := open(t, filepath.Join(dir, path))
		for i, ticker := range []string{"PETR4", "VALE3", "ITUB4"} {
			if _, err := store.Append(
				ctx, "acct-1", app.Any(), envelopeAt(t, i+1), domain.KindObservation, claim(t, ticker),
			); err != nil {
				t.Fatal(err)
			}
		}
		if err := store.Close(); err != nil {
			t.Fatal(err)
		}
		return encodedFacts(t, filepath.Join(dir, path))
	}

	original := write("first.db")
	replayed := write("second.db")

	if len(original) != len(replayed) {
		t.Fatalf("replay produced %d facts, want %d", len(replayed), len(original))
	}
	for i := range original {
		if string(original[i]) != string(replayed[i]) {
			t.Errorf("fact %d does not replay byte-identically; a content address would move", i)
		}
	}
}

// Case 14. Crash-safety, which ADR-0035 named as the gap its audit did not
// close and the thing it would want tested before this is trusted with a
// ledger.
//
// A full power-loss test is not portable. What is testable, and is the property
// that matters, is that a transaction which does not commit leaves nothing
// behind — and that the durability settings are the ones asserted rather than
// whatever the driver defaults to.
func TestAnInterruptedAppendLeavesNoPartialFact(t *testing.T) {
	path := filepath.Join(t.TempDir(), "ledger.db")
	store := open(t, path)

	// Land one fact so the stream exists and has something to be corrupted.
	if _, err := store.Append(
		context.Background(), "acct-1", app.Any(), envelopeAt(t, 1), domain.KindObservation, claim(t, "PETR4"),
	); err != nil {
		t.Fatal(err)
	}

	// Cancel the context before the append can commit.
	cancelled, cancel := context.WithCancel(context.Background())
	cancel()
	_, err := store.Append(
		cancelled, "acct-1", app.Any(), envelopeAt(t, 2), domain.KindObservation, claim(t, "VALE3"))
	if err == nil {
		t.Fatal("an append on a cancelled context reported success")
	}

	stream, err := store.Load(context.Background(), "acct-1")
	if err != nil {
		t.Fatalf("load after the interrupted append: %v", err)
	}
	if stream.Len() != 1 {
		t.Fatalf("the interrupted append left %d facts, want 1", stream.Len())
	}
	if err := store.Close(); err != nil {
		t.Fatal(err)
	}

	// The durability settings are asserted, not assumed (ADR-0035).
	raw := openRaw(t, path)
	defer func() { _ = raw.Close() }()
	for _, tc := range []struct{ pragma, want string }{
		{"journal_mode", "wal"},
		{"synchronous", "2"}, // 2 == FULL
	} {
		var got string
		if err := raw.QueryRow("PRAGMA " + tc.pragma).Scan(&got); err != nil {
			t.Fatalf("read %s: %v", tc.pragma, err)
		}
		if got != tc.want {
			t.Errorf("%s is %q, want %q — a committed fact may not survive power loss", tc.pragma, got, tc.want)
		}
	}
}

// Case 15. The sequence is unique at the storage layer, not only in Go.
//
// The in-process guarantee is that Append assigns it under a transaction. This
// asserts the schema would refuse a duplicate anyway, so a second writer — or a
// future bug — cannot produce two facts with one Ref.
func TestTheSchemaRefusesADuplicateSequence(t *testing.T) {
	path := filepath.Join(t.TempDir(), "ledger.db")
	store := open(t, path)
	if _, err := store.Append(
		context.Background(), "acct-1", app.Any(), envelopeAt(t, 1), domain.KindObservation, claim(t, "PETR4"),
	); err != nil {
		t.Fatal(err)
	}
	if err := store.Close(); err != nil {
		t.Fatal(err)
	}

	raw := openRaw(t, path)
	defer func() { _ = raw.Close() }()
	_, err := raw.Exec(
		`INSERT INTO facts (stream, sequence, effective_from, knowledge, encoded) VALUES ('acct-1', 1, 'x', 'y', x'00')`)
	if err == nil {
		t.Fatal("the schema accepted a second fact at sequence 1; a Ref would address two facts")
	}
}

// A deleted interior row is corruption, not a shape to tolerate. This is the
// scenario the 2026-08-07 audit measured: with the sequence derived from
// COUNT(*), deleting row 2 of 3 made Load return the third fact under ref #2 —
// silently re-pointing every later ref at different content — and made the next
// Append collide with the surviving row, leaving the stream permanently
// unappendable.
//
// An append-only stream whose sequence is assigned by the store (ADR-0034) has
// no legitimate source of gaps, so both paths must refuse rather than repair.
func TestAGapIsRefusedRatherThanRenumbered(t *testing.T) {
	path := filepath.Join(t.TempDir(), "ledger.db")
	store := open(t, path)
	ctx := context.Background()
	// A slice, not a map: knowledge time must increase with every append, and Go
	// randomises map iteration order — the exact nondeterminism this repository's
	// analysers exist to forbid, and it made this test flaky before it was fixed.
	for hour, ticker := range []string{1: "PETR4", 2: "VALE3", 3: "ITUB4"}[1:] {
		if _, err := store.Append(
			ctx, "acct-1", app.Any(), envelopeAt(t, hour+1), domain.KindObservation, claim(t, ticker),
		); err != nil {
			t.Fatalf("seed %d: %v", hour, err)
		}
	}
	if err := store.Close(); err != nil {
		t.Fatal(err)
	}

	raw := openRaw(t, path)
	if _, err := raw.Exec(`DELETE FROM facts WHERE stream = 'acct-1' AND sequence = 2`); err != nil {
		t.Fatal(err)
	}
	if err := raw.Close(); err != nil {
		t.Fatal(err)
	}

	reopened := open(t, path)

	t.Run("Load refuses", func(t *testing.T) {
		stream, err := reopened.Load(ctx, "acct-1")
		if !errors.Is(err, sqlitestore.ErrGap) {
			t.Fatalf("Load returned (%d facts, %v); want ErrGap — renumbering makes a "+
				"FactCorrected name a different fact than the one it corrects", stream.Len(), err)
		}
	})

	t.Run("Append refuses", func(t *testing.T) {
		_, err := reopened.Append(
			ctx, "acct-1", app.Any(), envelopeAt(t, 4), domain.KindObservation, claim(t, "BBAS3"))
		if !errors.Is(err, sqlitestore.ErrGap) {
			t.Fatalf("Append returned %v; want ErrGap rather than a UNIQUE constraint "+
				"failure or a compounded gap", err)
		}
	})
}

// The sequence must come from what was stored, not from a count of what
// survives. Load is asserted to reproduce the refs Append handed out, which is
// the property a FactCorrected reference depends on.
func TestLoadReproducesTheStoredSequences(t *testing.T) {
	path := filepath.Join(t.TempDir(), "ledger.db")
	store := open(t, path)
	ctx := context.Background()

	var appended []domain.Ref
	// A slice, not a map: knowledge time must increase with every append, and Go
	// randomises map iteration order — the exact nondeterminism this repository's
	// analysers exist to forbid, and it made this test flaky before it was fixed.
	for hour, ticker := range []string{1: "PETR4", 2: "VALE3", 3: "ITUB4"}[1:] {
		ref, err := store.Append(
			ctx, "acct-1", app.Any(), envelopeAt(t, hour+1), domain.KindObservation, claim(t, ticker))
		if err != nil {
			t.Fatalf("seed %d: %v", hour, err)
		}
		appended = append(appended, ref)
	}

	stream, err := store.Load(ctx, "acct-1")
	if err != nil {
		t.Fatal(err)
	}
	if stream.Len() != len(appended) {
		t.Fatalf("loaded %d facts, appended %d", stream.Len(), len(appended))
	}
	for _, want := range appended {
		fact, gErr := stream.Get(want)
		if gErr != nil {
			t.Errorf("ref %s is not resolvable after Load (%v); Append handed it to a caller", want, gErr)
			continue
		}
		if got := fact.Ref(); got != want {
			t.Errorf("Get(%s) returned a fact carrying ref %s", want, got)
		}
	}
}

// --- fixture -----------------------------------------------------------------

var epoch = time.Date(2026, time.March, 1, 0, 0, 0, 0, time.UTC)

func open(t *testing.T, path string) *sqlitestore.Store {
	t.Helper()
	store, err := sqlitestore.Open(path)
	if err != nil {
		t.Fatalf("open %s: %v", path, err)
	}
	t.Cleanup(func() { _ = store.Close() })
	return store
}

// openRaw reaches past the store to the database, which only a test that is
// asserting something *about* the storage layer may do.
func openRaw(t *testing.T, path string) *sql.DB {
	t.Helper()
	db, err := sql.Open("sqlite", path)
	if err != nil {
		t.Fatalf("open raw: %v", err)
	}
	return db
}

func encodedFacts(t *testing.T, path string) [][]byte {
	t.Helper()
	db := openRaw(t, path)
	defer func() { _ = db.Close() }()

	rows, err := db.Query(`SELECT encoded FROM facts WHERE stream = 'acct-1' ORDER BY sequence`)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = rows.Close() }()

	var out [][]byte
	for rows.Next() {
		var blob []byte
		if err := rows.Scan(&blob); err != nil {
			t.Fatal(err)
		}
		out = append(out, blob)
	}
	if err := rows.Err(); err != nil {
		t.Fatal(err)
	}
	return out
}

func claim(t *testing.T, ticker string) domain.HoldingClaimed {
	t.Helper()
	return domain.HoldingClaimed{
		Account:    identity.MustClaim("account_number", "0001234-5"),
		Instrument: identity.MustClaim("ticker", ticker),
		Quantity:   money.MustParseQuantity("100", "share"),
	}
}

func envelopeAt(t *testing.T, hour int) domain.Envelope {
	t.Helper()
	effective := temporal.MustAt(epoch)
	knowledge := temporal.MustAt(epoch.Add(time.Duration(hour) * time.Hour))

	interval, err := temporal.OpenFrom(effective)
	if err != nil {
		t.Fatal(err)
	}
	coordinates, err := temporal.Assign(interval, knowledge)
	if err != nil {
		t.Fatal(err)
	}
	source, err := provenance.NewSource(
		"sha256:0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef")
	if err != nil {
		t.Fatal(err)
	}
	prov, err := provenance.Observed(source, effective, provenance.Unmediated(), provenance.ConfidenceAsserted)
	if err != nil {
		t.Fatal(err)
	}
	envelope, err := domain.NewEnvelope(coordinates, prov, nil)
	if err != nil {
		t.Fatal(err)
	}
	return envelope
}

// A store written under the old encoding is refused, and the refusal names the
// migration (ADR-0040).
//
// Version zero is the case that needs care, and it is why Open reads the table
// as well as the pragma: a fresh file and a pre-ADR-0040 file both report zero,
// because the marker did not exist when the older one was written. What separates
// them is whether any fact is present.
//
// The alternative — opening it anyway — is not a smaller version of this. Because
// `CREATE TABLE IF NOT EXISTS` is a no-op against an existing file and a STRICT
// TEXT column silently coerces an integer, a mixed store would answer as-of
// queries wrongly, forever, with no error at any point.
func TestAStoreUnderTheOldEncodingIsRefused(t *testing.T) {
	path := filepath.Join(t.TempDir(), "legacy.db")

	// A pre-ADR-0040 store: TEXT temporal columns, one fact, no version marker.
	raw := openRaw(t, path)
	for _, stmt := range []string{
		`CREATE TABLE facts (
			stream TEXT NOT NULL, sequence INTEGER NOT NULL,
			effective_from TEXT NOT NULL, knowledge TEXT NOT NULL,
			encoded BLOB NOT NULL, PRIMARY KEY (stream, sequence)) STRICT`,
		`INSERT INTO facts VALUES ('acct-1', 1, '2026-03-01T00:00:00Z', '2026-03-01T01:00:00Z', x'00')`,
	} {
		if _, err := raw.Exec(stmt); err != nil {
			t.Fatal(err)
		}
	}
	var version int
	if err := raw.QueryRow(`PRAGMA user_version`).Scan(&version); err != nil {
		t.Fatal(err)
	}
	if version != 0 {
		t.Fatalf("fixture reports user_version %d; the case under test is zero", version)
	}
	if err := raw.Close(); err != nil {
		t.Fatal(err)
	}

	_, err := sqlitestore.Open(path)
	if !errors.Is(err, sqlitestore.ErrEncodingVersion) {
		t.Fatalf("Open returned %v; want ErrEncodingVersion — a store this build cannot order "+
			"must not answer an as-of query", err)
	}
	if !strings.Contains(err.Error(), "user_version") {
		t.Errorf("the refusal does not name the migration: %v", err)
	}
}

// An empty file at version zero is a *new* store, not a legacy one, and must
// open — otherwise the guard would refuse every first run.
func TestAFreshStoreClaimsTheCurrentEncoding(t *testing.T) {
	path := filepath.Join(t.TempDir(), "fresh.db")
	store := open(t, path)
	if err := store.Close(); err != nil {
		t.Fatal(err)
	}

	raw := openRaw(t, path)
	defer func() { _ = raw.Close() }()

	var version int
	if err := raw.QueryRow(`PRAGMA user_version`).Scan(&version); err != nil {
		t.Fatal(err)
	}
	if version != 2 {
		t.Errorf("a fresh store reports user_version %d, want 2; without the marker the next "+
			"build cannot tell this file from a legacy one", version)
	}

	// Reopening must be idempotent rather than tripping the guard.
	reopened := open(t, path)
	if err := reopened.Close(); err != nil {
		t.Fatal(err)
	}
}

// The columns hold integers, not renderings. Asserted through the stored type
// because that is the property the index depends on: a TEXT column would sort
// lexicographically however carefully the Go side compares.
func TestTemporalColumnsAreStoredAsIntegers(t *testing.T) {
	path := filepath.Join(t.TempDir(), "typed.db")
	store := open(t, path)
	if _, err := store.Append(
		context.Background(), "acct-1", app.Any(), envelopeAt(t, 1), domain.KindObservation, claim(t, "PETR4"),
	); err != nil {
		t.Fatal(err)
	}
	if err := store.Close(); err != nil {
		t.Fatal(err)
	}

	raw := openRaw(t, path)
	defer func() { _ = raw.Close() }()

	for _, column := range []string{"effective_from", "knowledge"} {
		var kind string
		if err := raw.QueryRow(
			`SELECT typeof(` + column + `) FROM facts WHERE stream = 'acct-1'`).Scan(&kind); err != nil {
			t.Fatal(err)
		}
		if kind != "integer" {
			t.Errorf("%s is stored as %s, want integer", column, kind)
		}
	}
}
