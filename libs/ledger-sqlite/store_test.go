package sqlite_test

import (
	"context"
	"database/sql"
	"path/filepath"
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
