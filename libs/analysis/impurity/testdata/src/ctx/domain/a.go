package domain

import (
	"context"       // want `impurity: import "context"`
	"encoding/json" // want `impurity: import "encoding/json"`
	"sync"          // want `impurity: import "sync"`
)

type Instrument struct {
	Symbol string `json:"symbol"` // want `impurity: serialisation tag "json"`
	Venue  string `db:"venue"`    // want `impurity: serialisation tag "db"`
}

var guard sync.Mutex

func Spawn() {
	go work() // want `impurity: goroutine`
}

func work() {}

func Emit(out chan int) { // want `impurity: channel type`
	out <- 1 // want `impurity: channel send`
}

func Consume(in <-chan int) int { // want `impurity: channel type`
	return <-in // want `impurity: channel receive`
}

func Race(a, b chan int) { // want `impurity: channel type`
	select { // want `impurity: select`
	case <-a: // want `impurity: channel receive`
	case <-b: // want `impurity: channel receive`
	}
}

func Fetch(ctx context.Context) error { // want `impurity: context\.Context`
	_ = ctx
	_, err := json.Marshal(struct{}{})
	return err
}
