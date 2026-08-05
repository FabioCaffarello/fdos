package kernel

// The shared kernel is as pure as a domain package. It holds Money, so the
// float ban matters here more than anywhere: a pattern matching only `domain`
// left this unchecked until M6.

type Amount float64 // want `nofloat: type float64`

var rate = 0.05 // want `nofloat: floating-point literal 0\.05`
