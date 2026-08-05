package domain

// Every construct below is what a financial domain package must not contain.

type Rate float64 // want `nofloat: type float64`

type Weight float32 // want `nofloat: type float32`

var price = 1.5 // want `nofloat: floating-point literal 1\.5`

func Scale(v float64) int { // want `nofloat: type float64`
	_ = v
	return 0
}

func Half(n int) int {
	// A literal is enough: it implies float arithmetic downstream.
	_ = 0.5 // want `nofloat: floating-point literal 0\.5`
	return n / 2
}

var rotation = 2i // want `nofloat: imaginary literal 2i`
