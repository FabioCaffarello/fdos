// Package scope decides which packages the FDOS determinism analysers apply to.
//
// The rules in this module exist to keep the domain layer a pure functional
// core (ADR-0013). They deliberately do NOT apply outside it: an adapter is
// supposed to call the clock, read the environment and start goroutines. An
// analyser that fired everywhere would be turned off, and a disabled rule
// enforces nothing.
package scope

import (
	"flag"
	"go/ast"
	"regexp"
	"strings"
	"sync"
)

// DefaultPattern matches the pure layers of the FDOS layout fixed by ADR-0013:
// the domain layer of any bounded context (libs/<context>/domain/...) and the
// shared kernel (libs/kernel/...).
//
// The kernel is included deliberately. It holds Money, and M6 found that a
// pattern matching only `domain` left the one package where binary floating
// point matters most entirely unchecked — the analysers would have passed a
// `float64` amount without a word.
const DefaultPattern = `(^|/)(domain|kernel)(/|$)`

var (
	mu      sync.RWMutex
	pattern = regexp.MustCompile(DefaultPattern)
)

// RegisterFlag adds -domain to an analyser's flag set, allowing the domain
// pattern to be overridden. The flag is shared state across analysers by
// design: a repository has one definition of "domain", and letting two
// analysers disagree about which packages are pure would be worse than the
// coupling.
func RegisterFlag(fs *flag.FlagSet) {
	fs.Func("domain", "regexp matching domain package paths (default `"+DefaultPattern+"`)", func(s string) error {
		re, err := regexp.Compile(s)
		if err != nil {
			return err
		}
		mu.Lock()
		defer mu.Unlock()
		pattern = re
		return nil
	})
}

// IsDomain reports whether a package path names a pure package.
//
// Synthesized test binaries (`<pkg>.test`) are excluded: they are generated
// scaffolding that imports `os` and `testing`, and reporting on them would mean
// reporting on code no human wrote.
func IsDomain(pkgPath string) bool {
	if strings.HasSuffix(pkgPath, ".test") {
		return false
	}
	mu.RLock()
	defer mu.RUnlock()
	return pattern.MatchString(pkgPath)
}

// IsGenerated reports whether a file carries the standard generated-code
// marker (https://go.dev/s/generatedcode).
//
// The purity rules do not apply to generated code, and M6 proved why by
// accident: `libs/contracts/gen/fdos/kernel/v1` matches the kernel pattern, so
// the analysers reported every protobuf message for `json` tags and `sync`.
//
// They were right on the substance — that is exactly ADR-0018's argument that
// generated wire types can never be domain types — but nobody wrote those
// files, so a report can only produce a suppression comment. The rules govern
// code a human is responsible for.
func IsGenerated(file *ast.File) bool {
	for _, group := range file.Comments {
		// The marker must precede the package clause to count.
		if group.Pos() > file.Package {
			break
		}
		for _, c := range group.List {
			line := strings.TrimSpace(strings.TrimPrefix(c.Text, "//"))
			if strings.HasPrefix(line, "Code generated ") && strings.HasSuffix(line, " DO NOT EDIT.") {
				return true
			}
		}
	}
	return false
}

// IsTestFile reports whether a filename is a Go test file.
//
// The purity rules do NOT apply to test files, and M6 is why. The property that
// justifies the entire decimal design — that folding the same amounts in a
// different order gives the same total — can only be tested by shuffling, and
// shuffling needs `math/rand`. A rule forbidding it in tests makes the property
// it protects impossible to prove.
//
// The rules exist to keep production domain code reproducible. Test files are
// not shipped, are not part of any derivation, and never reach the ledger.
// Applying the rules there buys nothing and costs the tests that matter most.
func IsTestFile(filename string) bool {
	return strings.HasSuffix(filename, "_test.go")
}
