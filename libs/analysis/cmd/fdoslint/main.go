// Command fdoslint enforces the FDOS domain purity rules.
//
// It is a composition root: it wires the analysers together and contains no
// logic of its own. The rules live in the sibling packages, where they are
// tested against both violating and compliant code.
//
//	go run ./cmd/fdoslint ./...
//
// Each analyser accepts a -domain regexp identifying which package paths are
// the pure domain layer. The default matches the layout fixed by ADR-0013.
package main

import (
	"golang.org/x/tools/go/analysis/multichecker"

	"github.com/FabioCaffarello/fdos/libs/analysis/impurity"
	"github.com/FabioCaffarello/fdos/libs/analysis/layering"
	"github.com/FabioCaffarello/fdos/libs/analysis/nofloat"
	"github.com/FabioCaffarello/fdos/libs/analysis/nondet"
)

func main() {
	multichecker.Main(
		nofloat.Analyzer,
		nondet.Analyzer,
		impurity.Analyzer,
		layering.Analyzer,
	)
}
