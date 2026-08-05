// Package identity assigns FDOS entity identifiers (RFC-0001, ADR-0007).
//
// Two properties that pull in opposite directions, and the design that gets
// both:
//
//   - **Deterministic.** Replaying the same input must produce the same
//     identifier, or no report is byte-reproducible (Constitution §9). So the
//     identifier is derived — a UUIDv5 over a namespace and a canonical seed —
//     not drawn at random.
//   - **Stable.** The identifier must survive the underlying facts changing. A
//     company renames, an ISIN is reassigned; the entity is the same entity.
//
// The resolution: the seed is a **birth certificate, not a key**. It fixes the
// identifier once, at first observation, and is never re-derived. Determinism
// comes from the derivation; stability comes from never repeating it.
//
// External identifiers are not identities. They are claims made by parties at
// points in time, and they belong in assertions, never here.
package identity

import (
	"crypto/sha1" //nolint:gosec // UUIDv5 is defined over SHA-1; not a security use
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
)

// Errors returned by this package.
var (
	// ErrEmptySeed is returned when a seed carries no distinguishing content.
	ErrEmptySeed = errors.New("identity: seed is empty")

	// ErrUnknownKind is returned for an entity kind outside the closed set.
	ErrUnknownKind = errors.New("identity: unknown entity kind")
)

// Kind is what an identifier identifies.
//
// A closed set, unlike identifier schemes. Kinds correspond to the aggregates
// fixed by ADR-0007; adding one is a modelling decision that should require
// review, where adding an identifier scheme must not (a private connector
// supporting a new institution cannot amend this package).
type Kind uint8

// Entity kinds. The zero value is invalid.
const (
	KindUnspecified Kind = iota
	KindInstrument
	KindParty
	KindAccount
	KindLedgerStream
)

// String returns the kind name, which is also its namespace component.
func (k Kind) String() string {
	switch k {
	case KindInstrument:
		return "instrument"
	case KindParty:
		return "party"
	case KindAccount:
		return "account"
	case KindLedgerStream:
		return "ledger_stream"
	default:
		return "unspecified"
	}
}

// Valid reports whether the kind is one of the defined ones.
func (k Kind) Valid() bool { return k >= KindInstrument && k <= KindLedgerStream }

// ID is an opaque internal identifier.
//
// Opaque means opaque: nothing parses it, and no meaning may be recovered from
// its structure. Code that reads structure out of an ID has created a coupling
// that the next identifier scheme breaks.
type ID struct {
	kind  Kind
	value string
}

// namespace is the FDOS root, itself a UUIDv5 over a fixed string. Constant so
// that identifier derivation is reproducible across processes and years.
var namespace = mustParseUUID("6ba7b810-9dad-11d1-80b4-00c04fd430c8")

// Derive assigns the identifier for an entity at first observation.
//
// `seed` is the canonicalised natural key as first seen — an ISIN, an account
// number, whatever distinguished the entity when FDOS met it. It is used once
// and never again: if the ISIN is later reassigned or the company renames, the
// identifier does not move, because it was never a function of current state.
//
// Callers must not use Derive to *look up* an entity. Resolution goes through
// recorded identifier assertions (ADR-0007), so that a wrong match is a fact
// that can be retracted rather than a computation that cannot be undone.
func Derive(kind Kind, seed string) (ID, error) {
	if !kind.Valid() {
		return ID{}, fmt.Errorf("%w: %d", ErrUnknownKind, kind)
	}
	canonical := canonicaliseSeed(seed)
	if canonical == "" {
		return ID{}, ErrEmptySeed
	}
	return ID{kind: kind, value: uuidV5(namespace, kind.String()+":"+canonical)}, nil
}

// MustDerive is Derive for constants and tests.
func MustDerive(kind Kind, seed string) ID {
	id, err := Derive(kind, seed)
	if err != nil {
		panic(err)
	}
	return id
}

// Kind returns what this identifier identifies.
func (id ID) Kind() Kind { return id.kind }

// String returns the opaque identifier.
func (id ID) String() string { return id.value }

// IsZero reports whether the identifier is unset.
func (id ID) IsZero() bool { return id.value == "" }

// Equal reports whether two identifiers denote the same entity.
func (id ID) Equal(other ID) bool { return id.kind == other.kind && id.value == other.value }

// canonicaliseSeed normalises a natural key before derivation.
//
// Versioned by behaviour rather than by a version field: changing this function
// changes the identifiers assigned to *new* entities, which is why it must not
// change casually. Existing identifiers are unaffected — they were assigned
// once and are never re-derived.
func canonicaliseSeed(seed string) string {
	return strings.ToUpper(strings.Join(strings.Fields(seed), " "))
}

// uuidV5 implements RFC 4122 §4.3: a name-based UUID over SHA-1.
//
// Deterministic by construction. No clock, no randomness — the `nondet`
// analyser would reject either in this package.
func uuidV5(ns [16]byte, name string) string {
	h := sha1.New() //nolint:gosec // UUIDv5 is defined over SHA-1
	h.Write(ns[:])
	h.Write([]byte(name))
	sum := h.Sum(nil)

	var out [16]byte
	copy(out[:], sum[:16])
	out[6] = (out[6] & 0x0f) | 0x50 // version 5
	out[8] = (out[8] & 0x3f) | 0x80 // RFC 4122 variant

	hexed := hex.EncodeToString(out[:])
	return hexed[0:8] + "-" + hexed[8:12] + "-" + hexed[12:16] + "-" + hexed[16:20] + "-" + hexed[20:32]
}

func mustParseUUID(s string) [16]byte {
	cleaned := strings.ReplaceAll(s, "-", "")
	raw, err := hex.DecodeString(cleaned)
	if err != nil || len(raw) != 16 {
		panic("identity: malformed namespace UUID")
	}
	var out [16]byte
	copy(out[:], raw)
	return out
}
