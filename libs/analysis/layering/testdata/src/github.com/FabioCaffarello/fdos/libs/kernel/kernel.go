package kernel

import (
	// The kernel is the shared vocabulary and may depend on no bounded context.
	portfolio "github.com/FabioCaffarello/fdos/libs/portfolio/domain" // want `layering: .* the kernel is the shared vocabulary`
)

// Money stands in for the canonical primitives (RFC-0002).
type Money struct{ Units int64 }

var _ portfolio.Allocation
