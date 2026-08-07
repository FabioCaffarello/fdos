package sqlite_test

// SPIKE — DELETE BEFORE MERGE. Not a test: a measurement that reports through
// a deliberate failure, because `make test` does not pass -v and a passing
// test's t.Logf output never reaches the CI log.
//
// It exists to answer one question the M11 gate could not answer on darwin:
// how much of the admission rejection rate is this platform's coarse wall
// clock, and how much is structural?
//
// Measured on darwin/arm64: 32 concurrent handlers → 3 admitted; with a clock
// handing out distinct strictly-increasing instants → 4 admitted; a sequential
// loop → between 4 and 32 admitted, run to run.

import (
	"context"
	"errors"
	"path/filepath"
	"runtime"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/FabioCaffarello/fdos/libs/kernel/identity"
	"github.com/FabioCaffarello/fdos/libs/kernel/money"
	"github.com/FabioCaffarello/fdos/libs/kernel/provenance"
	"github.com/FabioCaffarello/fdos/libs/kernel/temporal"
	"github.com/FabioCaffarello/fdos/libs/ledger/adapters/clock"
	"github.com/FabioCaffarello/fdos/libs/ledger/app"
)

// distinctInstants removes clock resolution from the experiment: every caller
// gets an instant no other caller got. A rejection that survives this is caused
// by the read and the append being reordered, not by two readings colliding.
type distinctInstants struct {
	base time.Time
	n    atomic.Int64
}

func (d *distinctInstants) Now() temporal.Instant {
	return temporal.MustAt(d.base.Add(time.Duration(d.n.Add(1)) * time.Millisecond))
}

func spikeCommand(t *testing.T) app.AcceptHoldingClaimCommand {
	t.Helper()
	at := temporal.MustAt(time.Date(2026, time.March, 1, 12, 0, 0, 0, time.UTC))
	src, err := provenance.NewSource(
		"sha256:0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef")
	if err != nil {
		t.Fatal(err)
	}
	effective, err := temporal.OpenFrom(at)
	if err != nil {
		t.Fatal(err)
	}
	return app.AcceptHoldingClaimCommand{
		Stream:      "acct-1",
		Account:     identity.MustClaim("account", "A-1"),
		Instrument:  identity.MustClaim("ticker", "PETR4"),
		Quantity:    money.MustParseQuantity("100", "share"),
		Effective:   effective,
		Source:      src,
		CollectedAt: at,
		Interpreter: provenance.Unmediated(),
		Confidence:  provenance.ConfidenceAsserted,
	}
}

func spikeSubmit(t *testing.T, knowledge app.Clock, handlers int, concurrent bool) (admitted, nonMono, stale int) {
	t.Helper()
	ctx := context.Background()
	store := open(t, filepath.Join(t.TempDir(), "ledger.db"))
	ledger, err := app.NewLedger(store, knowledge, identity.Canonicalisation())
	if err != nil {
		t.Fatal(err)
	}
	cmd := spikeCommand(t)

	count := func(err error) {
		switch {
		case err == nil:
			admitted++
		case errors.Is(err, app.ErrNonMonotonicKnowledge):
			nonMono++
		case errors.Is(err, app.ErrStaleRead):
			stale++
		default:
			t.Fatalf("unexpected: %v", err)
		}
	}

	if !concurrent {
		for i := 0; i < handlers; i++ {
			_, submitErr := ledger.AcceptHoldingClaim(ctx, cmd)
			count(submitErr)
		}
		return admitted, nonMono, stale
	}

	var wg sync.WaitGroup
	var mu sync.Mutex
	start := make(chan struct{})
	for i := 0; i < handlers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			<-start // released together, as a transport would
			_, submitErr := ledger.AcceptHoldingClaim(ctx, cmd)
			mu.Lock()
			defer mu.Unlock()
			count(submitErr)
		}()
	}
	close(start)
	wg.Wait()
	return admitted, nonMono, stale
}

func TestSpikeAdmissionUnderLoad(t *testing.T) {
	const handlers = 32
	const rounds = 5

	base := time.Date(2026, time.March, 1, 0, 0, 0, 0, time.UTC)

	report := "\n===== M11 SPIKE: admission under load =====\n"
	report += "platform: " + runtime.GOOS + "/" + runtime.GOARCH +
		"  NumCPU=" + itoa(runtime.NumCPU()) + "  GOMAXPROCS=" + itoa(runtime.GOMAXPROCS(0)) + "\n"

	// How coarse is a knowledge-time reading here? 10k readings, count distinct.
	seen := map[int64]struct{}{}
	var sys clock.System
	for i := 0; i < 10000; i++ {
		seen[sys.Now().Time().UnixNano()] = struct{}{}
	}
	report += "clock: 10000 readings -> " + itoa(len(seen)) + " distinct instants\n"
	report += "clock: three raw readings: " +
		itoa64(sys.Now().Time().UnixNano()) + " " +
		itoa64(sys.Now().Time().UnixNano()) + " " +
		itoa64(sys.Now().Time().UnixNano()) + "\n"

	for r := 1; r <= rounds; r++ {
		a1, n1, s1 := spikeSubmit(t, clock.System{}, handlers, true)
		a2, n2, s2 := spikeSubmit(t, &distinctInstants{base: base}, handlers, true)
		a3, n3, s3 := spikeSubmit(t, clock.System{}, handlers, false)
		report += "round " + itoa(r) +
			"  concurrent/wall:     admitted=" + itoa(a1) + " nonMono=" + itoa(n1) + " stale=" + itoa(s1) +
			"\n         concurrent/distinct: admitted=" + itoa(a2) + " nonMono=" + itoa(n2) + " stale=" + itoa(s2) +
			"\n         sequential/wall:     admitted=" + itoa(a3) + " nonMono=" + itoa(n3) + " stale=" + itoa(s3) + "\n"
	}
	report += "===== END SPIKE =====\n"

	// Deliberate failure: this is how the numbers reach the CI log.
	t.Error(report)
}

func itoa(n int) string { return itoa64(int64(n)) }

func itoa64(n int64) string {
	if n == 0 {
		return "0"
	}
	var buf [24]byte
	i := len(buf)
	for n > 0 {
		i--
		buf[i] = byte('0' + n%10)
		n /= 10
	}
	return string(buf[i:])
}
