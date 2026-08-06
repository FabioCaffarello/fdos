package memory_test

import (
	"testing"

	"github.com/FabioCaffarello/fdos/libs/ledger/adapters/memory"
	"github.com/FabioCaffarello/fdos/libs/ledger/app"
	"github.com/FabioCaffarello/fdos/libs/ledger/storetest"
)

// The in-memory store is held to the same definition of `app.Store` as a
// durable one (ADR-0034).
//
// The cases that used to be written out here by hand moved into
// `storetest` unchanged in substance. Keeping a private copy would have meant
// two definitions of what implementing the port means, and the drifted one is
// always the one that gets read.
//
// This store is not durable and does not claim to be. Reopening a database and
// finding the facts still there is an adapter obligation, and it belongs to the
// adapter that can satisfy it.
func TestConformance(t *testing.T) {
	storetest.Run(t, func(*testing.T) app.Store { return memory.NewStore() })
}
