package adapters

// Identical constructs outside the domain layer. None is reported: adapters
// legitimately deal in wire formats and third-party numeric types.

type Rate float64

type Weight float32

var price = 1.5

func Scale(v float64) float64 {
	return v * 2.0
}
