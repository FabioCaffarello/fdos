package adapters

import (
	"math/rand"
	"os"
	"time"
)

// An adapter is exactly where the clock, the environment and entropy are
// supposed to be read. Nothing here is reported.

func Stamp() time.Time {
	return time.Now()
}

func Config() string {
	return os.Getenv("FDOS_MODE")
}

func Draw() int {
	return rand.Int()
}

func Total(m map[string]int) int {
	total := 0
	for _, v := range m {
		total += v
	}
	return total
}
