package main

import (
	"context"
	"fmt"
	"time"

	"google.golang.org/protobuf/proto"

	ingestv1 "github.com/FabioCaffarello/fdos/libs/contracts/gen/fdos/ingest/v1"
	"github.com/FabioCaffarello/fdos/libs/kernel/identity"
	"github.com/FabioCaffarello/fdos/libs/kernel/temporal"
	ledgerwire "github.com/FabioCaffarello/fdos/libs/ledger-wire"
	"github.com/FabioCaffarello/fdos/libs/ledger/adapters/clock"
	"github.com/FabioCaffarello/fdos/libs/ledger/adapters/memory"
	"github.com/FabioCaffarello/fdos/libs/ledger/app"
)

// Check reports whether a serialized submission would be admitted.
//
// # It restates no rules, and that is the design
//
// A kit that re-implemented the admission rules would drift from them, and the
// drift has a direction: the kit passes something admission rejects. A producer
// then builds against a green kit and learns the truth from a rejection in
// production, which is worse than having no kit.
//
// So this does not describe admission — it *runs* it, against a throwaway
// in-memory ledger. The answer cannot disagree with the real one because it is
// produced by the same code.
//
// # What this is not
//
// It is not permission. The ledger revalidates every submission it receives,
// assuming nothing about what the caller ran, because a producer can link a
// modified build of anything published here (ADR-0029). Passing this is evidence
// about your submission, never a commitment from the ledger.
//
// The ledger it runs is *a* ledger, not *the* ledger: an empty in-memory store
// and a fixed clock, discarded when the call returns. Nothing is appended
// anywhere that outlives the check.
func Check(wire []byte) error {
	var submission ingestv1.HoldingClaimSubmission
	if err := proto.Unmarshal(wire, &submission); err != nil {
		return fmt.Errorf("not a HoldingClaimSubmission: %w", err)
	}

	cmd, err := ledgerwire.DecodeHoldingClaimSubmission(&submission)
	if err != nil {
		return fmt.Errorf("decode: %w", err)
	}

	at, err := temporal.At(time.Unix(0, 0).UTC())
	if err != nil {
		return fmt.Errorf("clock: %w", err)
	}
	ledger, err := app.NewLedger(
		memory.NewStore(), clock.NewSequence(at, time.Hour), identity.Canonicalisation())
	if err != nil {
		return fmt.Errorf("ledger: %w", err)
	}

	if _, err := ledger.AcceptHoldingClaim(context.Background(), cmd); err != nil {
		return err
	}
	return nil
}
