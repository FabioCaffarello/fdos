package nondet_test

import (
	"testing"

	"golang.org/x/tools/go/analysis/analysistest"

	"github.com/FabioCaffarello/fdos/libs/analysis/nondet"
)

func TestDomainPackagesRejectNonDeterministicInputs(t *testing.T) {
	analysistest.Run(t, analysistest.TestData(), nondet.Analyzer, "ctx/domain")
}

// Adapters are where the clock, environment and entropy belong. The rule must
// not fire there, or it will be disabled and enforce nothing.
func TestAdapterPackagesAreUnaffected(t *testing.T) {
	analysistest.Run(t, analysistest.TestData(), nondet.Analyzer, "ctx/adapters")
}
