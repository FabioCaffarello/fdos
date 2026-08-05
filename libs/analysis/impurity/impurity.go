// Package impurity forbids I/O and concurrency constructs in domain packages.
//
// The domain layer is a pure functional core (ADR-0013): data in, facts and
// decisions out. A pure core has no shared mutable state, so mutexes and
// channels protect nothing — their presence means I/O has been placed in the
// wrong layer.
//
// context.Context is treated the same way. It is an I/O concern, and RFC-0003
// forbids it in the domain specifically so that port interfaces cannot be
// declared there: a port with a ctx parameter would drag the whole imperative
// world back across the boundary.
//
// Serialisation is also excluded. A JSON tag on a canonical model is provider
// representation leaking into the domain, which is what Constitution §3 exists
// to prevent.
package impurity

import (
	"go/ast"
	"go/token"
	"go/types"
	"strings"

	"golang.org/x/tools/go/analysis"

	"github.com/FabioCaffarello/fdos/libs/analysis/internal/scope"
)

const doc = `check for I/O, concurrency and serialisation in domain packages

Reports goroutines, channels, select, sync primitives, context.Context and
serialisation. Each indicates code that belongs in the app or adapters layer
(ADR-0013, RFC-0003).`

// Analyzer is the impurity pass, for use with singlechecker, multichecker or
// go vet -vettool.
var Analyzer = &analysis.Analyzer{
	Name: "impurity",
	Doc:  doc,
	Run:  run,
}

func init() {
	scope.RegisterFlag(&Analyzer.Flags)
}

var forbiddenImports = map[string]string{
	"sync":          "a pure core has no shared mutable state to protect",
	"sync/atomic":   "a pure core has no shared mutable state to protect",
	"context":       "an I/O concern; ports belong in the app layer (RFC-0003)",
	"encoding/json": "serialisation is an adapter concern (Constitution §3)",
	"encoding/xml":  "serialisation is an adapter concern (Constitution §3)",
	"encoding/gob":  "serialisation is an adapter concern (Constitution §3)",
	"net":           "the domain performs no I/O (Constitution §10)",
	"net/http":      "the domain performs no I/O (Constitution §10)",
	"database/sql":  "the domain performs no I/O (Constitution §10)",
	"os":            "the domain performs no I/O (Constitution §10)",
	"io":            "the domain performs no I/O (Constitution §10)",
}

func run(pass *analysis.Pass) (any, error) {
	if !scope.IsDomain(pass.Pkg.Path()) {
		return nil, nil
	}

	for _, file := range pass.Files {
		if scope.IsTestFile(pass.Fset.Position(file.Pos()).Filename) || scope.IsGenerated(file) {
			continue
		}
		checkImports(pass, file)

		ast.Inspect(file, func(n ast.Node) bool {
			switch node := n.(type) {
			case *ast.GoStmt:
				pass.Reportf(node.Pos(),
					"impurity: goroutine in domain package; concurrency belongs in the app or adapters layer (ADR-0013)")

			case *ast.SelectStmt:
				pass.Reportf(node.Pos(),
					"impurity: select in domain package; concurrency belongs in the app or adapters layer (ADR-0013)")

			case *ast.SendStmt:
				pass.Reportf(node.Pos(),
					"impurity: channel send in domain package; a pure core communicates by return value (ADR-0013)")

			case *ast.ChanType:
				pass.Reportf(node.Pos(),
					"impurity: channel type in domain package; a pure core communicates by return value (ADR-0013)")

			case *ast.UnaryExpr:
				if node.Op == token.ARROW {
					pass.Reportf(node.Pos(),
						"impurity: channel receive in domain package; a pure core communicates by return value (ADR-0013)")
				}

			case *ast.StructType:
				checkTags(pass, node)

			case *ast.Ident:
				checkContextType(pass, node)
			}
			return true
		})
	}
	return nil, nil
}

func checkImports(pass *analysis.Pass, file *ast.File) {
	for _, spec := range file.Imports {
		path := strings.Trim(spec.Path.Value, `"`)
		if reason, forbidden := forbiddenImports[path]; forbidden {
			pass.Reportf(spec.Pos(),
				"impurity: import %q in domain package; %s", path, reason)
		}
	}
}

// checkTags reports serialisation struct tags. A canonical model carrying a
// json tag has had a wire format decided for it by whoever wrote the tag, which
// is exactly the provider leakage Constitution §3 forbids.
func checkTags(pass *analysis.Pass, st *ast.StructType) {
	if st.Fields == nil {
		return
	}
	for _, field := range st.Fields.List {
		if field.Tag == nil {
			continue
		}
		tag := field.Tag.Value
		for _, key := range []string{"json:", "xml:", "yaml:", "db:", "bson:", "protobuf:"} {
			if strings.Contains(tag, key) {
				pass.Reportf(field.Tag.Pos(),
					"impurity: serialisation tag %q on a domain type; the wire format belongs to adapters (Constitution §3)",
					strings.TrimSuffix(key, ":"))
				break
			}
		}
	}
}

// checkContextType catches context.Context appearing as a type even where the
// context package is imported under an alias or reached indirectly.
func checkContextType(pass *analysis.Pass, ident *ast.Ident) {
	obj, ok := pass.TypesInfo.Uses[ident].(*types.TypeName)
	if !ok || obj.Pkg() == nil {
		return
	}
	if obj.Pkg().Path() == "context" && obj.Name() == "Context" {
		pass.Reportf(ident.Pos(),
			"impurity: context.Context in domain package; an I/O concern — ports belong in the app layer (RFC-0003, ADR-0013)")
	}
}
