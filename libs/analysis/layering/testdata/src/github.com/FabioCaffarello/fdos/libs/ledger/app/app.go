package app

// Leaf: the inversion case is exercised from ledger/domain, so this package
// must not import ledger/domain or the graph would cycle.
type Recorder struct{}
