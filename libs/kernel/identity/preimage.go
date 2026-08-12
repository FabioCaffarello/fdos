package identity

import (
	"encoding/binary"
	"strings"
)

// The framed pre-image, accepted by ADR-0040.
//
// Everything hashed or compared in this package is built by [frame], never by
// joining strings with a separator. The rule it exists to enforce:
//
//	No two distinct component lists produce one byte string.
//
// A separator cannot promise that, because no separator is safe when a component
// may contain it — and escaping only moves the problem to the escape character.
// The measured failure: `scheme + ":" + value` gave `claim("ticker", "x:y")` and
// `claim("ticker:x", "y")` the same rendering, so two different claims derived
// one EntityId. A silent merge, in an append-only ledger, which is exactly what
// ADR-0007 forbids.
//
// The construction is Git's, which addresses `"blob " + len + "\x00" + content`
// for the same reason, and DER's discipline of enumerating the restrictions
// rather than asserting canonicality.

// tag names what a pre-image is for, so two different kinds of thing cannot
// collide even if their components are identical.
//
// A tag is a compile-time constant in this package and never caller-supplied.
// Adding one is an ADR-class act for the same reason changing [canonicaliseSeed]
// is: it changes the identifiers assigned to new entities.
type tag string

const (
	// tagEntitySeed frames a (kind, seed) pair — the generic path, used by
	// [Derive] for a seed that is not a claim, such as a ledger stream name.
	tagEntitySeed tag = "fdos.identity.entity-seed.v1"

	// tagClaimSeed frames a (kind, scheme, value) triple — the claim path, used
	// by [DeriveFromClaim] and [Ruleset.CanonicalSeed]. Distinct from
	// tagEntitySeed so that a claim can never collide with a bare seed that
	// happens to render the same way.
	tagClaimSeed tag = "fdos.identity.claim-seed.v1"
)

// frame renders a tag and its components as an injective byte string.
//
// Each component is preceded by its length as a fixed-width big-endian uint64.
// Fixed width because a variable-length count is itself ambiguous; big-endian so
// that the encoding sorts the way the numbers do, which costs nothing here and
// is the property the storage encodings need.
//
// The tag is framed like any other component rather than trusted to be
// unambiguous, so a future tag that is a prefix of another cannot collide.
func frame(t tag, components ...string) string {
	var b strings.Builder

	// The exact size, so the builder never reallocates: eight bytes of length
	// per component plus the tag, and then the bytes themselves.
	size := 8 + len(t)
	for _, c := range components {
		size += 8 + len(c)
	}
	b.Grow(size)

	write := func(s string) {
		var length [8]byte
		binary.BigEndian.PutUint64(length[:], uint64(len(s)))
		b.Write(length[:])
		b.WriteString(s)
	}

	write(string(t))
	for _, c := range components {
		write(c)
	}
	return b.String()
}
