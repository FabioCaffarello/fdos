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
	"regexp"
	"sync"
)

// DefaultPattern matches the domain layer of any bounded context, per the
// package layout fixed by ADR-0013: libs/<context>/domain/...
const DefaultPattern = `(^|/)domain(/|$)`

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

// IsDomain reports whether a package path names a domain package.
func IsDomain(pkgPath string) bool {
	mu.RLock()
	defer mu.RUnlock()
	return pattern.MatchString(pkgPath)
}
