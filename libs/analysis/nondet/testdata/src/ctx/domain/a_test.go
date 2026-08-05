package domain

import (
	"math/rand"
	"testing"
	"time"
)

// The purity rules do not apply to test files. Proving that a fold is
// order-independent requires shuffling, and shuffling requires randomness — a
// rule forbidding it here would make the property it protects untestable.
//
// Nothing in this file is reported.
func TestShufflingIsAllowedInTests(t *testing.T) {
	r := rand.New(rand.NewSource(1))
	xs := []int{1, 2, 3}
	r.Shuffle(len(xs), func(i, j int) { xs[i], xs[j] = xs[j], xs[i] })

	_ = time.Now()

	m := map[string]int{"a": 1}
	for k, v := range m {
		_, _ = k, v
	}
}
