// Package nondet forbids hidden, non-reproducible inputs in domain packages.
//
// Constitution §2 requires every business rule to be deterministic and every
// calculation reproducible. A calculation that reads the clock, draws random
// numbers or consults the environment cannot be reproduced from the ledger
// alone, because part of its input was never recorded.
//
// Clocks and entropy are not forbidden in FDOS — they are injected at the
// application boundary, where they become explicit parameters and therefore
// recordable provenance (RFC-0004, ADR-0010).
package nondet

import (
	"go/ast"
	"go/types"

	"golang.org/x/tools/go/analysis"

	"github.com/FabioCaffarello/fdos/libs/analysis/internal/scope"
)

const doc = `check for non-deterministic inputs in domain packages

Reports clock reads, randomness, environment access and map-iteration order
dependence. These make a calculation unreproducible from the ledger alone
(Constitution §2, §9).`

// Analyzer is the nondet pass, for use with singlechecker, multichecker or
// go vet -vettool.
var Analyzer = &analysis.Analyzer{
	Name: "nondet",
	Doc:  doc,
	Run:  run,
}

func init() {
	scope.RegisterFlag(&Analyzer.Flags)
}

// forbiddenPackages are non-deterministic in their entirety.
var forbiddenPackages = map[string]string{
	"math/rand":    "randomness must be injected, so the draw is recorded as provenance",
	"math/rand/v2": "randomness must be injected, so the draw is recorded as provenance",
	"crypto/rand":  "randomness must be injected, so the draw is recorded as provenance",
}

// forbiddenFuncs are specific non-deterministic entry points in packages that
// are otherwise legitimate. time.Time arithmetic is fine; reading the clock is
// not.
var forbiddenFuncs = map[string]map[string]string{
	"time": {
		"Now":   "the clock is injected; a domain rule that reads it cannot be replayed",
		"Since": "derives from time.Now",
		"Until": "derives from time.Now",
	},
	"os": {
		"Getenv":    "hidden input; a calculation must be reproducible from the ledger alone",
		"LookupEnv": "hidden input; a calculation must be reproducible from the ledger alone",
		"Environ":   "hidden input; a calculation must be reproducible from the ledger alone",
	},
}

func run(pass *analysis.Pass) (any, error) {
	if !scope.IsDomain(pass.Pkg.Path()) {
		return nil, nil
	}

	for _, file := range pass.Files {
		ast.Inspect(file, func(n ast.Node) bool {
			switch node := n.(type) {
			case *ast.CallExpr:
				checkCall(pass, node)

			case *ast.RangeStmt:
				checkRange(pass, node)
			}
			return true
		})
	}
	return nil, nil
}

func checkCall(pass *analysis.Pass, call *ast.CallExpr) {
	sel, ok := call.Fun.(*ast.SelectorExpr)
	if !ok {
		return
	}
	fn, ok := pass.TypesInfo.Uses[sel.Sel].(*types.Func)
	if !ok || fn.Pkg() == nil {
		return
	}

	pkgPath := fn.Pkg().Path()

	if reason, forbidden := forbiddenPackages[pkgPath]; forbidden {
		pass.Reportf(call.Pos(),
			"nondet: %s.%s in domain package; %s (Constitution §2)",
			pkgPath, fn.Name(), reason)
		return
	}

	if funcs, known := forbiddenFuncs[pkgPath]; known {
		if reason, forbidden := funcs[fn.Name()]; forbidden {
			pass.Reportf(call.Pos(),
				"nondet: %s.%s in domain package; %s (Constitution §2)",
				pkgPath, fn.Name(), reason)
		}
	}
}

// checkRange reports iteration over a map. Go randomises map iteration order
// deliberately, so any fold over a map is order-dependent unless the keys are
// sorted first. In a pure domain this is indistinguishable from a
// reproducibility bug, and it is invisible in testing because it usually
// happens to be consistent for small maps.
func checkRange(pass *analysis.Pass, rng *ast.RangeStmt) {
	t := pass.TypesInfo.TypeOf(rng.X)
	if t == nil {
		return
	}
	if _, isMap := t.Underlying().(*types.Map); !isMap {
		return
	}
	// Ranging purely for side effects, with neither key nor value bound, cannot
	// depend on order.
	if rng.Key == nil && rng.Value == nil {
		return
	}
	// The remedy for this rule is itself a map range: collect the keys, sort
	// them, then iterate the sorted slice. Reporting that idiom would make the
	// rule impossible to satisfy without a suppression comment, and a rule that
	// requires routine suppression is a rule that will be switched off.
	if isKeyCollectionIdiom(rng) {
		return
	}
	pass.Reportf(rng.Pos(),
		"nondet: range over map in domain package; Go randomises map iteration order — collect the keys, sort them, then iterate (Constitution §2, §9)")
}

// isKeyCollectionIdiom recognises exactly:
//
//	for k := range m {
//		keys = append(keys, k)
//	}
//
// Binding only the key, with a single-statement body that appends that key to a
// slice, cannot itself depend on iteration order — the order is decided by the
// sort that follows. Anything wider than this shape is reported.
func isKeyCollectionIdiom(rng *ast.RangeStmt) bool {
	if rng.Key == nil || rng.Value != nil {
		return false
	}
	key, ok := rng.Key.(*ast.Ident)
	if !ok {
		return false
	}
	if rng.Body == nil || len(rng.Body.List) != 1 {
		return false
	}
	assign, ok := rng.Body.List[0].(*ast.AssignStmt)
	if !ok || len(assign.Rhs) != 1 {
		return false
	}
	call, ok := assign.Rhs[0].(*ast.CallExpr)
	if !ok || len(call.Args) < 2 {
		return false
	}
	fn, ok := call.Fun.(*ast.Ident)
	if !ok || fn.Name != "append" {
		return false
	}
	last, ok := call.Args[len(call.Args)-1].(*ast.Ident)
	return ok && last.Name == key.Name
}
