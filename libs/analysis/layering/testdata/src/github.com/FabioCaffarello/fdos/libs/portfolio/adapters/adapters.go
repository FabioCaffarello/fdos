package adapters

// Adapters sit outermost and may import both inner layers of their own context.
import (
	portfolioapp "github.com/FabioCaffarello/fdos/libs/portfolio/app"
	portfoliodomain "github.com/FabioCaffarello/fdos/libs/portfolio/domain"
)

type Store struct {
	Allocator portfolioapp.Allocator
	Last      portfoliodomain.Allocation
}
