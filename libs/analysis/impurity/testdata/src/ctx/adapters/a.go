package adapters

import (
	"context"
	"encoding/json"
	"sync"
)

// Adapters are where I/O, concurrency and serialisation belong. Nothing here
// is reported.

type Payload struct {
	Symbol string `json:"symbol"`
}

var guard sync.Mutex

func Spawn(ctx context.Context, out chan []byte) error {
	b, err := json.Marshal(Payload{Symbol: "X"})
	if err != nil {
		return err
	}
	go func() {
		select {
		case out <- b:
		case <-ctx.Done():
		}
	}()
	return nil
}
