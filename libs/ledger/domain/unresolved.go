package domain

import (
	"errors"

	"github.com/FabioCaffarello/fdos/libs/kernel/identity"
	"github.com/FabioCaffarello/fdos/libs/kernel/temporal"
)

// UnresolvedClaim is a claim that reached the ledger and resolves to no minted
// identity at a coordinate.
//
// It exists because a connector can publish faithfully into silence. Claims
// accumulate, no HoldingObserved is derived, and nothing says so — recorded as
// an open item under B-007 from the moment resolution was built, because the
// gap was visible before anything could hit it.
type UnresolvedClaim struct {
	// Fact is the claim fact that could not be resolved.
	Fact Ref

	// Claim is the specific identifier that did not resolve. One fact can
	// appear twice here: a holding names an account and an instrument, and
	// either may be unresolved independently of the other.
	Claim identity.Claim
}

// Unresolved lists every claim visible at a coordinate that resolves to no
// minted identity.
//
// Read-only and derives nothing. Answering "which claims are waiting" must not
// be the thing that stops them waiting: minting on inspection would make the
// act of looking change the ledger, and an identity would come into existence
// because somebody ran a report.
//
// Order follows the stream's, so the answer is deterministic and two calls at
// the same coordinate agree.
func Unresolved(stream Stream, asOf temporal.AsOf) []UnresolvedClaim {
	var out []UnresolvedClaim

	for _, f := range stream.VisibleAt(asOf) {
		claimed, ok := f.Payload().(HoldingClaimed)
		if !ok {
			continue
		}
		for _, c := range []identity.Claim{claimed.Account, claimed.Instrument} {
			if _, err := Resolve(stream, c, asOf); errors.Is(err, ErrUnresolved) {
				out = append(out, UnresolvedClaim{Fact: f.Ref(), Claim: c})
			}
		}
	}
	return out
}
