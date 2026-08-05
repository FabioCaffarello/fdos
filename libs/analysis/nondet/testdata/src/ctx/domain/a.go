package domain

import (
	"math/rand"
	"os"
	"sort"
	"time"
)

func Stamp() time.Time {
	return time.Now() // want `nondet: time\.Now`
}

func Elapsed(start time.Time) time.Duration {
	return time.Since(start) // want `nondet: time\.Since`
}

func Config() string {
	return os.Getenv("FDOS_MODE") // want `nondet: os\.Getenv`
}

func Draw() int {
	return rand.Int() // want `nondet: math/rand\.Int`
}

func Total(m map[string]int) int {
	total := 0
	for _, v := range m { // want `nondet: range over map`
		total += v
	}
	return total
}

// Deterministic arithmetic on an injected instant is fine: the clock was read
// at the boundary and recorded, not consulted here.
func Age(now, born time.Time) time.Duration {
	return now.Sub(born)
}

// The correct way to fold a map: collect the keys, sort them, then iterate.
// The collection loop is itself a map range and must NOT be reported, or the
// rule would be impossible to satisfy without a suppression comment.
func TotalSorted(m map[string]int) int {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	total := 0
	for _, k := range keys {
		total += m[k]
	}
	return total
}

// Binding only the key is not sufficient on its own: this body reads values in
// map order, so it is reported.
func FirstKeyed(m map[string]int) int {
	for k := range m { // want `nondet: range over map`
		return m[k]
	}
	return 0
}

// Appending something other than the key is not the idiom either.
func Values(m map[string]int) []int {
	out := make([]int, 0, len(m))
	for k := range m { // want `nondet: range over map`
		out = append(out, m[k])
	}
	return out
}
