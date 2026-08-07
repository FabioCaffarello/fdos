package main

import (
	"bytes"
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"google.golang.org/protobuf/proto"

	ingestv1 "github.com/FabioCaffarello/fdos/libs/contracts/gen/fdos/ingest/v1"
	kernelv1 "github.com/FabioCaffarello/fdos/libs/contracts/gen/fdos/kernel/v1"
	"github.com/FabioCaffarello/fdos/libs/kernel/identity"
	"github.com/FabioCaffarello/fdos/libs/kernel/money"
	"github.com/FabioCaffarello/fdos/libs/kernel/provenance"
	"github.com/FabioCaffarello/fdos/libs/kernel/temporal"
	sqlitestore "github.com/FabioCaffarello/fdos/libs/ledger-sqlite"
	ledgerwire "github.com/FabioCaffarello/fdos/libs/ledger-wire"
	"github.com/FabioCaffarello/fdos/libs/ledger/adapters/clock"
	"github.com/FabioCaffarello/fdos/libs/ledger/app"
	"github.com/FabioCaffarello/fdos/libs/ledger/domain"
)

const validDigest = "sha256:" + "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef"

// A producer that cannot mint an identity puts a fact in the ledger over a
// socket. That is the milestone, and everything below is about the ways it must
// refuse.
func TestASubmissionIsAdmitted(t *testing.T) {
	h, store := fixture(t)

	res := post(t, h, marshal(t, submission(t, validDigest)))
	if res.Code != http.StatusCreated {
		t.Fatalf("status %d, want 201; body %q", res.Code, res.Body.String())
	}
	ref := strings.TrimSpace(res.Body.String())
	if !strings.HasPrefix(ref, "acct-1#") {
		t.Errorf("returned reference %q, want acct-1#N", ref)
	}

	stream, err := store.Load(context.Background(), "acct-1")
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if stream.Len() != 1 {
		t.Errorf("the stream holds %d facts, want 1", stream.Len())
	}
}

// A Content-Type may carry parameters and still be the media type this service
// accepts. Comparing the header by string equality refuses a conformant client
// for a reason the message does not explain, which is the defect this asserts is
// gone.
func TestContentTypeParametersAreNotARefusal(t *testing.T) {
	for _, ct := range []string{
		"application/x-protobuf",
		"application/x-protobuf; charset=utf-8",
		"application/x-protobuf;charset=utf-8",
		"Application/X-Protobuf",
	} {
		t.Run(ct, func(t *testing.T) {
			h, _ := fixture(t)
			req := httptest.NewRequest(http.MethodPost, submissionPath,
				bytes.NewReader(marshal(t, submission(t, validDigest))))
			req.Header.Set("Content-Type", ct)
			res := httptest.NewRecorder()
			h.ServeHTTP(res, req)

			if res.Code != http.StatusCreated {
				t.Fatalf("%q was refused with %d: %s", ct, res.Code, res.Body.String())
			}
		})
	}
}

// Every one of these must be refused, and the status code must distinguish
// "you sent nonsense" from "the ledger will not have it" from "try again".
func TestRefusals(t *testing.T) {
	for _, tc := range []struct {
		name   string
		body   []byte
		method string
		ctype  string
		want   int
	}{
		{
			name: "not a protobuf message at all",
			body: []byte("this is not protobuf, it is a sentence"),
			want: http.StatusBadRequest,
		},
		{
			name:   "a GET",
			body:   marshal(t, submission(t, validDigest)),
			method: http.MethodGet,
			want:   http.StatusMethodNotAllowed,
		},
		{
			name:  "JSON, which is not the contract",
			body:  []byte(`{"stream":"acct-1"}`),
			ctype: "application/json",
			want:  http.StatusUnsupportedMediaType,
		},
		{
			name:  "no content type",
			body:  marshal(t, submission(t, validDigest)),
			ctype: "-",
			want:  http.StatusUnsupportedMediaType,
		},
		{
			name:  "a content type that is not a media type at all",
			body:  marshal(t, submission(t, validDigest)),
			ctype: ";;;",
			want:  http.StatusUnsupportedMediaType,
		},
		{
			name:  "the right media type inside a longer string",
			body:  marshal(t, submission(t, validDigest)),
			ctype: "text/plain, application/x-protobuf",
			want:  http.StatusUnsupportedMediaType,
		},
		{
			name: "a body over the bound",
			body: append(marshal(t, submission(t, validDigest)), make([]byte, maxBody)...),
			want: http.StatusRequestEntityTooLarge,
		},
		{
			name: "provenance that is a URL rather than a content address",
			body: marshal(t, submission(t, "https://broker.example/statement.pdf")),
			want: http.StatusUnprocessableEntity,
		},
		{
			name: "provenance that is an account id",
			body: marshal(t, submission(t, "account-88213")),
			want: http.StatusUnprocessableEntity,
		},
		{
			name: "no effective interval",
			body: marshal(t, withoutEffective(t)),
			want: http.StatusUnprocessableEntity,
		},
		{
			name: "an empty message",
			body: marshal(t, &ingestv1.HoldingClaimSubmission{}),
			want: http.StatusUnprocessableEntity,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			h, store := fixture(t)

			req := httptest.NewRequest(orDefault(tc.method, http.MethodPost),
				submissionPath, bytes.NewReader(tc.body))
			switch tc.ctype {
			case "":
				req.Header.Set("Content-Type", protobufMediaType)
			case "-":
				// deliberately absent
			default:
				req.Header.Set("Content-Type", tc.ctype)
			}
			res := httptest.NewRecorder()
			h.ServeHTTP(res, req)

			if res.Code != tc.want {
				t.Fatalf("status %d, want %d; body %q", res.Code, tc.want, res.Body.String())
			}

			// The refusal must also mean nothing was written. A service that
			// answers 4xx after appending is worse than one that appends.
			if _, err := store.Load(context.Background(), "acct-1"); !errors.Is(err, app.ErrStreamNotFound) {
				t.Errorf("a refused submission left something in the ledger: %v", err)
			}
		})
	}
}

// A 5xx is a statement about this process, made to an unauthenticated caller —
// D2 is open, so that is every caller. It must not carry the internals.
func TestAFailureDoesNotLeakInternals(t *testing.T) {
	ledger, err := app.NewLedger(
		failingStore{errors.New("sqlite: /var/lib/fdos/ledger.db: disk I/O error near line 42")},
		clock.System{}, identity.Canonicalisation())
	if err != nil {
		t.Fatal(err)
	}

	res := post(t, newHandler(ledger), marshal(t, submission(t, validDigest)))
	if res.Code != http.StatusInternalServerError {
		t.Fatalf("status %d, want 500", res.Code)
	}
	for _, leaked := range []string{"sqlite", "/var/lib", "disk I/O", "line 42"} {
		if strings.Contains(res.Body.String(), leaked) {
			t.Errorf("the 500 body leaked %q: %s", leaked, res.Body.String())
		}
	}
}

// ADR-0030's decision is an absence, and an absence is only enforced for as long
// as nobody adds the field. This is the test that notices.
//
// Not a transport concern, and deliberately here anyway: this is the process
// that would carry a producer-supplied knowledge time if one ever existed, so
// this is where its arrival should break something.
func TestASubmissionCannotCarryKnowledgeTime(t *testing.T) {
	fields := (&ingestv1.HoldingClaimSubmission{}).ProtoReflect().Descriptor().Fields()
	for i := 0; i < fields.Len(); i++ {
		name := string(fields.Get(i).Name())
		if strings.Contains(name, "knowledge") || strings.Contains(name, "known") {
			t.Errorf("HoldingClaimSubmission carries %q — a producer must not supply "+
				"knowledge time (ADR-0009, ADR-0030)", name)
		}
	}
}

// The end-to-end half of ADR-0036: concurrent callers over the transport, one
// stream, nothing refused.
//
// The unit-level property is tested in libs/ledger/app. This one exists because
// it is the only place that proves the composition root did not undo it — by
// constructing a Ledger per request, which is the mistake NewLedger's doc warns
// about and which no type can catch.
func TestConcurrentSubmissionsAreAllAdmitted(t *testing.T) {
	const callers = 32

	h, store := fixture(t)
	body := marshal(t, submission(t, validDigest))

	var wg sync.WaitGroup
	codes := make(chan int, callers)
	start := make(chan struct{})
	for i := 0; i < callers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			<-start
			codes <- post(t, h, body).Code
		}()
	}
	close(start)
	wg.Wait()
	close(codes)

	refused := 0
	for code := range codes {
		if code != http.StatusCreated {
			refused++
		}
	}
	if refused > 0 {
		t.Errorf("%d of %d concurrent submissions were refused", refused, callers)
	}

	stream, err := store.Load(context.Background(), "acct-1")
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if stream.Len() != callers {
		t.Errorf("the stream holds %d facts after %d submissions", stream.Len(), callers)
	}
}

// The default must not expose this process, and exposing it must be an act.
func TestListenAddressRefusesToBeExposedByAccident(t *testing.T) {
	for _, tc := range []struct {
		name         string
		addr         string
		acknowledged bool
		wantErr      bool
	}{
		{name: "the default", addr: "127.0.0.1:8080"},
		{name: "loopback by name is still an address", addr: "127.0.0.2:8080"},
		{name: "IPv6 loopback", addr: "[::1]:8080"},
		{name: "every interface", addr: ":8080", wantErr: true},
		{name: "every interface, spelled out", addr: "0.0.0.0:8080", wantErr: true},
		{name: "a routable address", addr: "10.0.0.5:8080", wantErr: true},
		{name: "a hostname, which may resolve anywhere", addr: "ledger.internal:8080", wantErr: true},
		{name: "every interface, acknowledged", addr: "0.0.0.0:8080", acknowledged: true},
		{name: "not an address at all", addr: "8080", wantErr: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			err := checkListenAddress(tc.addr, tc.acknowledged)
			if tc.wantErr && err == nil {
				t.Fatalf("%q was accepted without acknowledgement", tc.addr)
			}
			if !tc.wantErr && err != nil {
				t.Fatalf("%q was refused: %v", tc.addr, err)
			}
		})
	}
}

// The default in the flag definition is the thing an operator gets by doing
// nothing, so it is the thing worth asserting.
func TestTheDefaultConfigurationIsLoopback(t *testing.T) {
	cfg, err := parse([]string{"-store", "ledger.db"}, nil)
	if err != nil {
		t.Fatalf("a minimal invocation was refused: %v", err)
	}
	if cfg.listen != "127.0.0.1:8080" {
		t.Errorf("the default listen address is %q", cfg.listen)
	}
	if cfg.acknowledged {
		t.Error("the acknowledgement defaults to true, which defeats it")
	}
	if _, err := parse([]string{"-listen", "0.0.0.0:8080", "-store", "l.db"}, nil); err == nil {
		t.Error("a public listen address was accepted with no acknowledgement")
	}
	if _, err := parse(nil, nil); err == nil {
		t.Error("a missing -store was accepted; there is no default database path")
	}
}

// --- fixtures ----------------------------------------------------------------

func fixture(t *testing.T) (http.Handler, *sqlitestore.Store) {
	t.Helper()
	store, err := sqlitestore.Open(filepath.Join(t.TempDir(), "ledger.db"))
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	t.Cleanup(func() { _ = store.Close() })

	ledger, err := app.NewLedger(store, clock.System{}, identity.Canonicalisation())
	if err != nil {
		t.Fatalf("ledger: %v", err)
	}
	return newHandler(ledger), store
}

func post(t *testing.T, h http.Handler, body []byte) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(http.MethodPost, submissionPath, bytes.NewReader(body))
	req.Header.Set("Content-Type", protobufMediaType)
	res := httptest.NewRecorder()
	h.ServeHTTP(res, req)
	return res
}

func submission(t *testing.T, source string) *ingestv1.HoldingClaimSubmission {
	t.Helper()
	at := temporal.MustAt(time.Date(2026, time.March, 1, 12, 0, 0, 0, time.UTC))
	src, err := provenance.NewSource(source)
	if err != nil {
		// A deliberately malformed source is the subject of several cases, and
		// the encoder is what must carry it through to admission unchanged.
		return rawSubmission(t, source)
	}
	effective, err := temporal.OpenFrom(at)
	if err != nil {
		t.Fatal(err)
	}
	return ledgerwire.EncodeHoldingClaimSubmission(app.AcceptHoldingClaimCommand{
		Stream:      "acct-1",
		Account:     identity.MustClaim("account", "A-1"),
		Instrument:  identity.MustClaim("ticker", "PETR4"),
		Quantity:    money.MustParseQuantity("100", "share"),
		Effective:   effective,
		Source:      src,
		CollectedAt: at,
		Interpreter: provenance.Unmediated(),
		Confidence:  provenance.ConfidenceAsserted,
	})
}

// rawSubmission builds a well-formed message carrying a source the kernel would
// have refused to construct — which is exactly what a hostile producer sends,
// and what admission must catch on the ledger's side of the boundary.
func rawSubmission(t *testing.T, source string) *ingestv1.HoldingClaimSubmission {
	t.Helper()
	valid := submission(t, validDigest)
	valid.Source = &kernelv1.SourceRef{Value: source}
	return valid
}

func withoutEffective(t *testing.T) *ingestv1.HoldingClaimSubmission {
	t.Helper()
	s := submission(t, validDigest)
	s.Effective = nil
	return s
}

func marshal(t *testing.T, m *ingestv1.HoldingClaimSubmission) []byte {
	t.Helper()
	b, err := proto.Marshal(m)
	if err != nil {
		t.Fatal(err)
	}
	return b
}

func orDefault(v, fallback string) string {
	if v == "" {
		return fallback
	}
	return v
}

// failingStore is the only substitute in these tests, and it substitutes for
// storage rather than for a decision.
type failingStore struct{ err error }

func (f failingStore) Load(context.Context, string) (domain.Stream, error) {
	return domain.Stream{}, f.err
}

func (f failingStore) Append(
	context.Context, string, app.Expectation, domain.Envelope, domain.Kind, domain.Payload,
) (domain.Ref, error) {
	return domain.Ref{}, f.err
}
