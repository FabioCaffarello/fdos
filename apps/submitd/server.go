package main

import (
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"

	"google.golang.org/protobuf/proto"

	ingestv1 "github.com/FabioCaffarello/fdos/libs/contracts/gen/fdos/ingest/v1"
	"github.com/FabioCaffarello/fdos/libs/kernel/provenance"
	"github.com/FabioCaffarello/fdos/libs/ledger/app"
	"github.com/FabioCaffarello/fdos/libs/ledger/domain"
	ledgerwire "github.com/FabioCaffarello/fdos/libs/ledger-wire"
)

// submissionPath is the one route this service has.
const submissionPath = "/v1/holding-claim-submissions"

// protobufMediaType is the only body encoding accepted.
//
// The published message rather than a JSON re-expression of it, so a producer in
// any language sends the bytes the conformance kit produces (ADR-0018,
// ADR-0037).
const protobufMediaType = "application/x-protobuf"

// maxBody bounds a request body at 1 MiB.
//
// A submission is a handful of fields; a megabyte is three orders of magnitude
// of headroom. The bound exists because the caller is unauthenticated — D2 is
// open — so an unbounded read is an unbounded allocation from a stranger.
const maxBody = 1 << 20

// newHandler builds the router.
//
// It takes the concrete *app.Ledger rather than an interface. An interface here
// would exist only to let a test substitute something that admits differently,
// which is the shape of a service that decides admission — and this one does
// not (ADR-0037 §2).
func newHandler(ledger *app.Ledger) http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc(submissionPath, func(w http.ResponseWriter, r *http.Request) {
		submit(w, r, ledger)
	})
	return mux
}

func submit(w http.ResponseWriter, r *http.Request, ledger *app.Ledger) {
	if r.Method != http.MethodPost {
		w.Header().Set("Allow", http.MethodPost)
		refuse(w, http.StatusMethodNotAllowed, "a submission is a POST")
		return
	}
	if ct := r.Header.Get("Content-Type"); ct != protobufMediaType {
		refuse(w, http.StatusUnsupportedMediaType,
			fmt.Sprintf("body must be %s, got %q", protobufMediaType, ct))
		return
	}

	// One byte past the limit distinguishes "exactly at the bound" from "over
	// it"; without it a body of exactly maxBody is indistinguishable from a
	// truncated larger one.
	body, err := io.ReadAll(io.LimitReader(r.Body, maxBody+1))
	if err != nil {
		refuse(w, http.StatusBadRequest, "could not read the body")
		return
	}
	if len(body) > maxBody {
		refuse(w, http.StatusRequestEntityTooLarge,
			fmt.Sprintf("a submission is at most %d bytes", maxBody))
		return
	}

	var wire ingestv1.HoldingClaimSubmission
	if err := proto.Unmarshal(body, &wire); err != nil {
		refuse(w, http.StatusBadRequest, "body is not a fdos.ingest.v1.HoldingClaimSubmission")
		return
	}

	cmd, err := ledgerwire.DecodeHoldingClaimSubmission(&wire)
	if err != nil {
		// The message parsed as protobuf and does not describe a submission —
		// a missing interval, an unparseable quantity. The producer needs to
		// know which, and it is a statement about what it sent rather than
		// about anything else in the ledger.
		refuse(w, http.StatusUnprocessableEntity, err.Error())
		return
	}

	ref, err := ledger.AcceptHoldingClaim(r.Context(), cmd)
	if err != nil {
		refuse(w, statusFor(err), reasonFor(err))
		return
	}

	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.WriteHeader(http.StatusCreated)
	// The reference is text because fdos.ledger.v1 publishes no Ref message and
	// adding one is a contract change rather than a handler convenience.
	_, _ = fmt.Fprintln(w, ref.String())
}

// statusFor maps a refusal from the ledger onto a status code.
//
// The mapping is the whole of this service's judgement, and it is about
// transport rather than about admission: which of these the ledger returns is
// entirely the ledger's decision.
func statusFor(err error) int {
	switch {
	case errors.Is(err, provenance.ErrMalformedSource),
		errors.Is(err, domain.ErrIncompleteEnvelope),
		errors.Is(err, domain.ErrUnknownKind):
		// Well-formed bytes the ledger refuses on their merits. The producer
		// can fix these and resend; nothing else in the ledger is involved.
		return http.StatusUnprocessableEntity
	case errors.Is(err, app.ErrStaleRead),
		errors.Is(err, app.ErrNonMonotonicKnowledge):
		// A genuine conflict with another writer. Inside one process ADR-0036
		// makes these unreachable for admission; across processes they are not,
		// and 409 says "the same request may succeed later" — which is true of
		// these and of nothing else here.
		return http.StatusConflict
	default:
		return http.StatusInternalServerError
	}
}

// reasonFor decides what the caller is told.
//
// A 4xx is a statement about the caller's own submission, so it carries the
// ledger's own words — the conformance kit is on the other end of this, and a
// producer that cannot see why it was refused cannot conform.
//
// A 5xx is not. It is a statement about this process, to an unauthenticated
// caller, and the text would carry file paths and driver internals. D2 is open,
// so "unauthenticated" is every caller.
func reasonFor(err error) string {
	if statusFor(err) >= http.StatusInternalServerError {
		return "the submission could not be recorded"
	}
	return err.Error()
}

func refuse(w http.ResponseWriter, status int, reason string) {
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.WriteHeader(status)
	_, _ = fmt.Fprintln(w, reason)
}

// errOffLoopbackUnacknowledged is returned rather than a bare string so that a
// test can assert on the refusal rather than on its wording.
var errOffLoopbackUnacknowledged = errors.New(
	"submitd: refusing to listen off-loopback without -callers-are-authenticated")

// checkListenAddress refuses an address that exposes this process to a network
// unless the operator has said they put authentication in front of it.
//
// **This is not an answer to D2** (fdos#64). It is a refusal to answer it by
// accident: a default reachable from anywhere would decide "anyone may write to
// any stream" silently, and that decision deserves an ADR rather than a flag
// default that grew into a policy (ADR-0037 §5).
func checkListenAddress(addr string, acknowledged bool) error {
	if acknowledged {
		return nil
	}
	host, _, err := net.SplitHostPort(addr)
	if err != nil {
		return fmt.Errorf("submitd: -listen must be host:port: %w", err)
	}
	// An empty host means every interface, which is the case this exists for and
	// the one that does not parse as an IP.
	if host == "" {
		return errOffLoopbackUnacknowledged
	}
	ip := net.ParseIP(host)
	if ip == nil {
		// A name, not an address. It may resolve anywhere, including off-host,
		// and resolving it here would make the answer depend on DNS at start-up.
		return errOffLoopbackUnacknowledged
	}
	if !ip.IsLoopback() {
		return errOffLoopbackUnacknowledged
	}
	return nil
}
