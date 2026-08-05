// Package layering enforces the module and layer topology fixed by ADR-0013.
//
// Two failures are worth preventing and neither is caught by the purity rules:
//
//   - **Layer inversion.** A domain package importing its own app or adapters
//     layer inverts the dependency the architecture is built on, and does it
//     without any of the constructs impurity looks for.
//   - **Cross-context coupling.** One bounded context importing another's
//     domain or app makes two ubiquitous languages one, which is the modelling
//     failure Constitution §3 and the Domain Vision exist to prevent. It looks
//     entirely innocent at the call site.
//
// Only packages matching the ADR-0013 layout are constrained. Tooling modules,
// technology adapter modules and composition roots in apps/ are unrestricted:
// composing everything is precisely what a composition root is for.
package layering

import (
	"flag"
	"fmt"
	"strings"

	"golang.org/x/tools/go/analysis"
)

const doc = `check module and layer boundaries

Reports layer inversion within a bounded context and imports that cross between
bounded contexts. The topology is fixed by ADR-0013.`

// DefaultPrefix is the module path prefix fixed by ADR-0003.
const DefaultPrefix = "github.com/FabioCaffarello/fdos/"

var prefix = DefaultPrefix

// Analyzer is the layering pass, for use with singlechecker, multichecker or
// go vet -vettool.
var Analyzer = &analysis.Analyzer{
	Name:  "layering",
	Doc:   doc,
	Run:   run,
	Flags: *flag.NewFlagSet("layering", flag.ExitOnError),
}

func init() {
	Analyzer.Flags.StringVar(&prefix, "prefix", DefaultPrefix,
		"module path prefix identifying first-party packages")
}

// location describes where a first-party package sits in the ADR-0013 topology.
type location struct {
	context string // bounded context name; "" for the kernel
	layer   string // domain | app | adapters
	kernel  bool
}

// classify reports where a package sits, and whether the topology constrains it
// at all. Anything not matching libs/kernel/... or libs/<context>/<layer>/... is
// unconstrained by design.
func classify(pkgPath string) (location, bool) {
	if !strings.HasPrefix(pkgPath, prefix) {
		return location{}, false
	}
	rest := strings.TrimPrefix(pkgPath, prefix)
	parts := strings.Split(rest, "/")
	if len(parts) < 2 || parts[0] != "libs" {
		return location{}, false
	}
	if parts[1] == "kernel" {
		return location{kernel: true}, true
	}
	if len(parts) < 3 {
		return location{}, false
	}
	switch parts[2] {
	case "domain", "app", "adapters":
		return location{context: parts[1], layer: parts[2]}, true
	}
	return location{}, false
}

// mayImport reports whether a package at `from` is permitted to import `to`,
// and if not, why.
func mayImport(from, to location) (bool, string) {
	// The kernel is the shared vocabulary. Everything may depend on it; it may
	// depend on nothing first-party, or it would couple every context to
	// whatever it reached for.
	if to.kernel {
		return true, ""
	}
	if from.kernel {
		return false, "the kernel is the shared vocabulary and must depend on no bounded context"
	}

	if from.context != to.context {
		return false, fmt.Sprintf(
			"bounded context %q must not reach into %q; contexts integrate through published contracts, not shared types (Constitution §3, §11)",
			from.context, to.context)
	}

	// Within one context, dependencies point inward only.
	rank := map[string]int{"domain": 0, "app": 1, "adapters": 2}
	if rank[to.layer] > rank[from.layer] {
		return false, fmt.Sprintf(
			"layer inversion: %s must not import %s; dependencies point inward (ADR-0013)",
			from.layer, to.layer)
	}
	return true, ""
}

func run(pass *analysis.Pass) (any, error) {
	from, constrained := classify(pass.Pkg.Path())
	if !constrained {
		return nil, nil
	}

	for _, file := range pass.Files {
		for _, spec := range file.Imports {
			path := strings.Trim(spec.Path.Value, `"`)
			to, isFirstParty := classify(path)
			if !isFirstParty {
				continue
			}
			if ok, reason := mayImport(from, to); !ok {
				pass.Reportf(spec.Pos(), "layering: import %q — %s", path, reason)
			}
		}
	}
	return nil, nil
}
