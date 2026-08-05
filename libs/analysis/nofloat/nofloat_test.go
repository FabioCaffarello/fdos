package nofloat_test

import (
	"testing"

	"golang.org/x/tools/go/analysis/analysistest"

	"github.com/FabioCaffarello/fdos/libs/analysis/nofloat"
)

func TestDomainPackagesRejectBinaryFloatingPoint(t *testing.T) {
	analysistest.Run(t, analysistest.TestData(), nofloat.Analyzer, "ctx/domain")
}

// The rule must NOT fire outside the domain layer. An analyser that reports
// everywhere gets disabled, and a disabled rule enforces nothing.
func TestAdapterPackagesAreUnaffected(t *testing.T) {
	analysistest.Run(t, analysistest.TestData(), nofloat.Analyzer, "ctx/adapters")
}

// The shared kernel is checked too. It holds Money (ADR-0008), so a pattern
// matching only `domain` would leave the one package where binary floating
// point matters most entirely unchecked.
func TestKernelIsCheckedLikeDomain(t *testing.T) {
	analysistest.Run(t, analysistest.TestData(), nofloat.Analyzer, "ctx/kernel")
}
