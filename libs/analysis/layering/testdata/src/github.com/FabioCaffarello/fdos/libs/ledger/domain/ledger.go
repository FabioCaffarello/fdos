package domain

import (
	"github.com/FabioCaffarello/fdos/libs/kernel"

	ledgerapp "github.com/FabioCaffarello/fdos/libs/ledger/app"           // want `layering: .* layer inversion: domain must not import app`
	ledgeradapters "github.com/FabioCaffarello/fdos/libs/ledger/adapters" // want `layering: .* layer inversion: domain must not import adapters`
	portfolio "github.com/FabioCaffarello/fdos/libs/portfolio/domain"     // want `layering: .* bounded context "ledger" must not reach into "portfolio"`
)

// Importing the kernel is the one first-party dependency the domain may have.
type Entry struct {
	Amount kernel.Money
}

var (
	_ ledgerapp.Recorder
	_ ledgeradapters.Store
	_ portfolio.Allocation
)
