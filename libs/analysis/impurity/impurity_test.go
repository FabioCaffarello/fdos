package impurity_test

import (
	"testing"

	"golang.org/x/tools/go/analysis/analysistest"

	"github.com/FabioCaffarello/fdos/libs/analysis/impurity"
)

func TestDomainPackagesRejectIOAndConcurrency(t *testing.T) {
	analysistest.Run(t, analysistest.TestData(), impurity.Analyzer, "ctx/domain")
}

// Adapters are where I/O, concurrency and serialisation belong.
func TestAdapterPackagesAreUnaffected(t *testing.T) {
	analysistest.Run(t, analysistest.TestData(), impurity.Analyzer, "ctx/adapters")
}

// Generated code is exempt. The kernel pattern matches
// libs/contracts/gen/fdos/kernel/v1, and M6 found the analysers reporting every
// protobuf message there. Correct on the substance, useless as a finding.
func TestGeneratedCodeIsExempt(t *testing.T) {
	analysistest.Run(t, analysistest.TestData(), impurity.Analyzer, "ctx/kernel")
}
