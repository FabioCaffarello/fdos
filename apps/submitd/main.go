// Command submitd receives claim submissions and admits them to the ledger.
//
// The first composition root in apps/ (ADR-0037). It reads configuration,
// constructs the store, the clock and the application service, wires them into
// an HTTP handler, and starts a process. That is the whole of its
// responsibility — every rule about what may be admitted lives in
// libs/ledger/app and is revalidated there whatever this binary did.
package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/FabioCaffarello/fdos/libs/kernel/identity"
	sqlitestore "github.com/FabioCaffarello/fdos/libs/ledger-sqlite"
	"github.com/FabioCaffarello/fdos/libs/ledger/adapters/clock"
	"github.com/FabioCaffarello/fdos/libs/ledger/app"
)

func main() {
	if err := run(os.Args[1:], os.Stderr); err != nil {
		fmt.Fprintf(os.Stderr, "submitd: %v\n", err)
		os.Exit(1)
	}
}

type config struct {
	listen       string
	store        string
	acknowledged bool
}

func parse(args []string, errOut *os.File) (config, error) {
	var c config
	fs := flag.NewFlagSet("submitd", flag.ContinueOnError)
	fs.SetOutput(errOut)
	fs.StringVar(&c.listen, "listen", "127.0.0.1:8080",
		"address to listen on; loopback unless -callers-are-authenticated is set")
	fs.StringVar(&c.store, "store", "", "path to the SQLite ledger database (required)")
	fs.BoolVar(&c.acknowledged, "callers-are-authenticated", false,
		"assert that authentication sits in front of this process; required to listen off-loopback")
	if err := fs.Parse(args); err != nil {
		return config{}, err
	}
	if c.store == "" {
		return config{}, errors.New("-store is required; there is no default database path")
	}
	return c, checkListenAddress(c.listen, c.acknowledged)
}

func run(args []string, errOut *os.File) error {
	cfg, err := parse(args, errOut)
	if err != nil {
		return err
	}

	store, err := sqlitestore.Open(cfg.store)
	if err != nil {
		return err
	}
	// Closing the database is the last thing that can lose a fact, so a failure
	// is reported rather than discarded. It cannot change the exit path — run
	// has already returned by then — which is exactly why it would otherwise
	// vanish.
	defer func() {
		if closeErr := store.Close(); closeErr != nil {
			slog.Error("the ledger database did not close cleanly", "error", closeErr)
		}
	}()

	// One Ledger for the process. It owns the lock table that serialises writes
	// per stream, so constructing one per request would silently give up the
	// property ADR-0036 exists for.
	ledger, err := app.NewLedger(store, clock.System{}, identity.Canonicalisation())
	if err != nil {
		return err
	}

	srv := &http.Server{
		Addr:              cfg.listen,
		Handler:           newHandler(ledger),
		ReadHeaderTimeout: 5 * time.Second,
	}

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	errs := make(chan error, 1)
	go func() {
		slog.Info("listening", "addr", cfg.listen, "store", cfg.store)
		if listenErr := srv.ListenAndServe(); listenErr != nil &&
			!errors.Is(listenErr, http.ErrServerClosed) {
			errs <- listenErr
		}
		close(errs)
	}()

	select {
	case err := <-errs:
		return err
	case <-ctx.Done():
	}

	// Drain in flight requests rather than dropping them. An admission that was
	// accepted and then lost to a shutdown is a fact the producer believes was
	// recorded and was not.
	shutdown, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	return srv.Shutdown(shutdown)
}
