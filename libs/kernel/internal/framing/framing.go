// Package framing is the one canonical encoding for values that will be hashed
// or ordered (ADR-0040).
//
// Internal, and shared rather than copied. ADR-0040 decided that "every value
// that will be hashed or ordered is serialised by a **single function**" — two
// copies drifting apart is the failure that decision exists to prevent, and a
// pre-image that differs between packages is a collision nobody would find by
// reading either one.
package framing

import (
	"encoding/binary"
	"strings"
)

// Frame renders a tag and its components as an injective byte string.
//
// The rule it enforces:
//
//	No two distinct (tag, components) pairs produce one byte string.
//
// Each component is preceded by its length as a fixed-width big-endian uint64.
// Fixed width because a variable-length count is itself ambiguous; big-endian so
// that the encoding sorts the way the numbers do, which costs nothing here and
// is the property a storage encoding needs.
//
// A separator cannot make the same promise, because no separator is safe when a
// component may contain it — and escaping only moves the problem to the escape
// character. The measured failures this replaces: `scheme + ":" + value` let a
// colon in a value impersonate the scheme boundary, and a derivation pre-image
// joined with "\nparam=" let a parameter value forge another parameter.
//
// The construction is Git's, which addresses `"blob " + len + "\x00" + content`
// for the same reason, and DER's discipline of enumerating the restrictions
// rather than asserting canonicality.
//
// The tag is framed like any other component rather than trusted to be
// unambiguous, so a future tag that is a prefix of another cannot collide.
//
// **Nesting is how structure is encoded.** A Frame result is a string, so a
// section of a larger pre-image is framed and passed as one component. That is
// what keeps grouping unambiguous: a flat list of components could not
// distinguish two inputs and no parameters from one input and one parameter,
// because the counts would not be part of the encoding.
func Frame(tag string, components ...string) string {
	var b strings.Builder

	// The exact size, so the builder never reallocates: eight bytes of length
	// per component plus the tag, and then the bytes themselves.
	size := 8 + len(tag)
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

	write(tag)
	for _, c := range components {
		write(c)
	}
	return b.String()
}
