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
