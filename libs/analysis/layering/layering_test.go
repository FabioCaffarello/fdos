package layering_test

import (
	"testing"

	"golang.org/x/tools/go/analysis/analysistest"

	"github.com/FabioCaffarello/fdos/libs/analysis/layering"
)

// The domain layer may import the kernel and nothing else first-party.
// Exercised on the `ledger` context, whose app and adapters are deliberate
// leaves so that the inversion imports do not create a package cycle.
func TestDomainRejectsInversionAndCrossContextImports(t *testing.T) {
	analysistest.Run(t, analysistest.TestData(), layering.Analyzer,
		"github.com/FabioCaffarello/fdos/libs/ledger/domain")
}

// The kernel is the shared vocabulary: it may depend on no bounded context.
func TestKernelMayNotImportABoundedContext(t *testing.T) {
	analysistest.Run(t, analysistest.TestData(), layering.Analyzer,
		"github.com/FabioCaffarello/fdos/libs/kernel")
}

// The permitted directions must stay silent. A rule that fires on the common
// case is a rule that gets disabled, and a disabled rule enforces nothing.
// Exercised on the `portfolio` context, which is layered correctly.
func TestAppMayImportOwnDomain(t *testing.T) {
	analysistest.Run(t, analysistest.TestData(), layering.Analyzer,
		"github.com/FabioCaffarello/fdos/libs/portfolio/app")
}

func TestAdaptersMayImportInnerLayers(t *testing.T) {
	analysistest.Run(t, analysistest.TestData(), layering.Analyzer,
		"github.com/FabioCaffarello/fdos/libs/portfolio/adapters")
}

// Tooling modules, technology adapter modules and composition roots are
// unconstrained by design — composing everything is what a composition root is
// for. This analyser's own module is the convenient example.
func TestToolingModulesAreUnconstrained(t *testing.T) {
	analysistest.Run(t, analysistest.TestData(), layering.Analyzer,
		"github.com/FabioCaffarello/fdos/libs/ledger/app")
}
