// Package nofloat forbids binary floating point in domain packages.
//
// This is the single highest-leverage rule in FDOS. The usual objection to
// float in financial code is representation error — 0.1 + 0.2 != 0.3 — but the
// decisive problem here is different and worse: floating-point addition is not
// associative. A projection that folds the same events in a different but
// equally valid order produces a different total.
//
// That breaks Constitution §9 on its own, independently of precision: the same
// ledger must yield a byte-identical report. Money and quantities use
// arbitrary-precision decimals (RFC-0002, ADR-0008).
package nofloat

import (
	"go/ast"
	"go/token"
	"go/types"

	"golang.org/x/tools/go/analysis"

	"github.com/FabioCaffarello/fdos/libs/analysis/internal/scope"
)

const doc = `check for binary floating-point types and literals in domain packages

Floating-point addition is not associative, so a fold in a different order
yields a different total. Financial values use arbitrary-precision decimals
(RFC-0002, ADR-0008).`

// Analyzer is the nofloat pass, for use with singlechecker, multichecker or
// go vet -vettool.
var Analyzer = &analysis.Analyzer{
	Name: "nofloat",
	Doc:  doc,
	Run:  run,
}

func init() {
	scope.RegisterFlag(&Analyzer.Flags)
}

func run(pass *analysis.Pass) (any, error) {
	if !scope.IsDomain(pass.Pkg.Path()) {
		return nil, nil
	}

	for _, file := range pass.Files {
		ast.Inspect(file, func(n ast.Node) bool {
			switch node := n.(type) {
			case *ast.BasicLit:
				switch node.Kind {
				case token.FLOAT:
					pass.Reportf(node.Pos(),
						"nofloat: floating-point literal %s in domain package; use a decimal (RFC-0002, ADR-0008)",
						node.Value)
				case token.IMAG:
					pass.Reportf(node.Pos(),
						"nofloat: imaginary literal %s in domain package; complex arithmetic has no place in a financial domain",
						node.Value)
				}

			case *ast.Ident:
				obj, ok := pass.TypesInfo.Uses[node].(*types.TypeName)
				if !ok {
					return true
				}
				basic, ok := obj.Type().(*types.Basic)
				if !ok {
					return true
				}
				if basic.Info()&(types.IsFloat|types.IsComplex) != 0 {
					pass.Reportf(node.Pos(),
						"nofloat: type %s in domain package; binary floating point is not associative, so fold order changes the result (RFC-0002, ADR-0008)",
						basic.Name())
				}
			}
			return true
		})
	}
	return nil, nil
}
