package app

// The permitted direction: app depends inward on its own domain. Nothing here
// is reported — a rule that fires on the common case gets disabled.
import portfoliodomain "github.com/FabioCaffarello/fdos/libs/portfolio/domain"

type Allocator struct {
	Target portfoliodomain.Allocation
}
