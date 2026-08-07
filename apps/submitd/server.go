package main

import (
	"errors"
	"fmt"
	"io"
	"log/slog"
	"mime"
	"net"
	"net/http"
	"strings"

	"google.golang.org/protobuf/proto"

	ingestv1 "github.com/FabioCaffarello/fdos/libs/contracts/gen/fdos/ingest/v1"
	"github.com/FabioCaffarello/fdos/libs/kernel/provenance"
	ledgerwire "github.com/FabioCaffarello/fdos/libs/ledger-wire"
	"github.com/FabioCaffarello/fdos/libs/ledger/app"
	"github.com/FabioCaffarello/fdos/libs/ledger/domain"
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

// isProtobuf reports whether the media type is the one this service accepts,
// ignoring any parameters.
//
// An absent or unparseable header is not the media type, which is the answer
// that keeps a caller who sent nothing from being treated as one who sent the
// right thing.
func isProtobuf(header string) bool {
	mediaType, _, err := mime.ParseMediaType(header)
	if err != nil {
		return false
	}
	return mediaType == protobufMediaType
}

func submit(w http.ResponseWriter, r *http.Request, ledger *app.Ledger) {
	if r.Method != http.MethodPost {
		w.Header().Set("Allow", http.MethodPost)
		refuse(w, http.StatusMethodNotAllowed, "a submission is a POST")
		return
	}
	// Parsed rather than compared. A Content-Type may carry parameters —
	// `application/x-protobuf; charset=utf-8` is the same media type — and a
	// string equality here refuses a conformant client for a reason it cannot
	// see from the message. The parameters are then ignored, because protobuf
	// has no charset and a client that sends one is wrong about something that
	// does not matter.
	if ct := r.Header.Get("Content-Type"); !isProtobuf(ct) {
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
	if unmarshalErr := proto.Unmarshal(body, &wire); unmarshalErr != nil {
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
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.WriteHeader(http.StatusCreated)
	// The reference is text because fdos.ledger.v1 publishes no Ref message and
	// adding one is a contract change rather than a handler convenience.
	write(w, ref.String())
}

// write emits the body and records a failure rather than discarding it.
//
// The fact is already appended by the time this runs, so a failed write is not
// a failed submission — but it does mean a producer believes its submission was
// lost when it was not, and it will resend. That is worth a line in a log and
// is not worth a 500, which would be a lie about what happened.
//
// Logged rather than ignored because errcheck runs with check-blank: a
// discarded error here would have to be spelled `_ =`, and the configuration
// deliberately refuses that spelling.
func write(w http.ResponseWriter, line string) {
	if _, err := io.WriteString(w, safe(line)+"\n"); err != nil {
		slog.Warn("the response could not be written; the fact is recorded regardless",
			"error", err)
	}
}

// safe bounds what a caller can make this service say back.
//
// A 4xx reason carries the ledger's own words, and those words quote the
// submission — a malformed SourceRef is echoed so the producer can see which
// one. So the response body contains caller-controlled bytes, and `gosec` is
// right to flag that as a taint path.
//
// Two things make it inert rather than one, because the cheap one is not
// enough on its own:
//
//   - printable ASCII only, bounded at 512 bytes. Control characters, newlines
//     and anything that could open a tag are dropped rather than escaped —
//     escaping preserves the payload for whatever decodes next.
//   - the caller gets `text/plain` with `X-Content-Type-Options: nosniff`, so
//     nothing reinterprets the body as markup.
//
// The bound matters separately: a submission may carry a long string, and an
// error message quoting it in full would let a caller choose the size of the
// response to an unauthenticated request.
func safe(reason string) string {
	const bound = 512

	var b strings.Builder
	for _, r := range reason {
		if b.Len() >= bound {
			b.WriteString("…")
			break
		}
		if r >= 0x20 && r < 0x7f && r != '<' && r != '>' && r != '&' {
			b.WriteRune(r)
			continue
		}
		b.WriteByte('.')
	}
	return b.String()
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
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.WriteHeader(status)
	write(w, reason)
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
