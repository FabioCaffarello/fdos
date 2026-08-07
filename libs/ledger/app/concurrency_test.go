package app_test

import (
	"context"
	"errors"
	"sync"
	"testing"

	"github.com/FabioCaffarello/fdos/libs/kernel/identity"
	"github.com/FabioCaffarello/fdos/libs/ledger/adapters/clock"
	"github.com/FabioCaffarello/fdos/libs/ledger/adapters/memory"
	"github.com/FabioCaffarello/fdos/libs/ledger/app"
	"github.com/FabioCaffarello/fdos/libs/ledger/domain"
)

// The property ADR-0036 exists for: concurrent admission to one stream loses
// nothing.
//
// Measured on linux/amd64 before the lock existed: 32 concurrent submissions,
// 3 to 9 admitted, the rest refused with ErrNonMonotonicKnowledge — and a clock
// handing every caller a distinct instant did not help, because the gap is
// between reading the clock and reaching the store rather than between two
// readings.
//
// # One Ledger, deliberately
//
// The lock table is a field of Ledger, so a test that constructed one per
// goroutine would pass while exercising nothing. That is the trap this test is
// written around, and the reason the sharing is stated here rather than left to
// be noticed.
//
// # The clock is the real one
//
// `clock.System` rather than a sequence that cannot collide. A sequence would
// still catch a missing lock — the appends would invert — but it would hide the
// second half of the property: that serialising is enough with a clock whose
// resolution is whatever the platform gives. Both readings matter, and only the
// real clock tests both.
func TestConcurrentAdmissionLosesNothing(t *testing.T) {
	const handlers = 64

	ledger, cmd := concurrentFixture(t)
	ctx := context.Background()

	var (
		wg    sync.WaitGroup
		mu    sync.Mutex
		refs  = make(map[domain.Ref]int, handlers)
		fails []error
		start = make(chan struct{})
	)

	for i := 0; i < handlers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			<-start // released together, as a transport would
			ref, err := ledger.AcceptHoldingClaim(ctx, cmd)
			mu.Lock()
			defer mu.Unlock()
			if err != nil {
				fails = append(fails, err)
				return
			}
			refs[ref]++
		}()
	}
	close(start)
	wg.Wait()

	if len(fails) > 0 {
		t.Errorf("%d of %d concurrent admissions were refused; first: %v",
			len(fails), handlers, fails[0])
	}
	if len(refs) != handlers-len(fails) {
		t.Errorf("%d distinct refs for %d successful appends — two appends shared a Ref",
			len(refs), handlers-len(fails))
	}
	for ref, n := range refs {
		if n > 1 {
			t.Errorf("%s was handed to %d callers", ref, n)
		}
	}
}

// Minting is the use case `Expectation` was built for, and the one where a race
// produces two identities for one claim rather than a rejected write.
//
// Concurrent mints of the same claim must produce exactly one mint and refuse
// the rest with ErrAlreadyMinted — never ErrStaleRead, which inside one process
// would mean the resolve-then-append sequence was not serialised.
func TestConcurrentMintsProduceOneIdentity(t *testing.T) {
	const handlers = 16

	ledger, ctx := mintOnlyFixture(t)
	claim := identity.MustClaim("ticker", "PETR4")
	cmd := mintCommand(t, identity.KindInstrument, claim, domain.Ref{})

	var (
		wg      sync.WaitGroup
		mu      sync.Mutex
		minted  []domain.Ref
		already int
		other   []error
		start   = make(chan struct{})
	)

	for i := 0; i < handlers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			<-start
			ref, err := ledger.MintIdentity(ctx, cmd)
			mu.Lock()
			defer mu.Unlock()
			switch {
			case err == nil:
				minted = append(minted, ref)
			case errors.Is(err, app.ErrAlreadyMinted):
				already++
			default:
				other = append(other, err)
			}
		}()
	}
	close(start)
	wg.Wait()

	if len(minted) != 1 {
		t.Errorf("%d mints for one claim, want exactly 1 (ADR-0033)", len(minted))
	}
	if already != handlers-len(minted) {
		t.Errorf("%d refusals were ErrAlreadyMinted, want %d", already, handlers-len(minted))
	}
	for _, err := range other {
		t.Errorf("unexpected failure — inside one process a stale read means the "+
			"resolve-then-append sequence was not serialised: %v", err)
	}
}

// The lock is per stream rather than global, so writes spread across streams
// must all land.
//
// Asserted by outcome rather than by timing: a timing assertion would be flaky
// on a loaded machine and would fail for reasons having nothing to do with the
// lock. What this catches is a shard collision that turned into a rejection,
// which is the failure mode a fixed lock table could plausibly introduce.
func TestWritesAcrossStreamsAllLand(t *testing.T) {
	const streams = 8
	const perStream = 8

	ledger, base := concurrentFixture(t)
	ctx := context.Background()

	var wg sync.WaitGroup
	errs := make(chan error, streams*perStream)
	for s := 0; s < streams; s++ {
		name := "acct-" + string(rune('a'+s))
		for i := 0; i < perStream; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				cmd := base
				cmd.Stream = name
				if _, err := ledger.AcceptHoldingClaim(ctx, cmd); err != nil {
					errs <- err
				}
			}()
		}
	}
	wg.Wait()
	close(errs)

	var failed int
	var first error
	for err := range errs {
		if failed == 0 {
			first = err
		}
		failed++
	}
	if failed > 0 {
		t.Errorf("%d of %d writes across %d streams were refused; first: %v",
			failed, streams*perStream, streams, first)
	}
}

// concurrentFixture builds a Ledger on the real clock.
//
// Separate from acceptFixture because that one injects a sequence clock, which
// is documented as unsafe for concurrent use — correctly, for the tests it
// serves.
func concurrentFixture(t *testing.T) (*app.Ledger, app.AcceptHoldingClaimCommand) {
	t.Helper()

	_, cmd := acceptFixture(t, validDigest)
	ledger, err := app.NewLedger(memory.NewStore(), clock.System{}, identity.Canonicalisation())
	if err != nil {
		t.Fatalf("ledger: %v", err)
	}
	return ledger, cmd
}
