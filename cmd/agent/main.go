// Command sourceant-agent keeps the local index running and serves it.
package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"github.com/sourceant/agent/internal/api"
	"github.com/sourceant/agent/internal/config"
	"github.com/sourceant/agent/internal/core"
	"github.com/sourceant/agent/internal/supervise"
)

// Set at build time. See the Makefile.
var (
	Version   = "dev"
	BuildTime = "unknown"
	GitCommit = "unknown"
)

func main() {
	if len(os.Args) > 1 && (os.Args[1] == "version" || os.Args[1] == "--version") {
		fmt.Printf("sourceant-agent %s (%s, built %s)\n", Version, GitCommit, BuildTime)
		return
	}
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, "sourceant-agent:", err)
		os.Exit(1)
	}
}

func run() error {
	cfg, err := config.FromEnvironment()
	if err != nil {
		return err
	}

	port := cfg.CorePort
	if port == 0 {
		if port, err = supervise.FreePort(); err != nil {
			return fmt.Errorf("finding a port for the core: %w", err)
		}
	}
	coreURL := "http://127.0.0.1:" + strconv.Itoa(port)
	client := core.New(coreURL, 30*time.Second)

	supervisor := supervise.New(supervise.Options{
		Name:        cfg.Core,
		Args:        []string{"serve", "--host", "127.0.0.1", "--port", strconv.Itoa(port)},
		Ready:       client.Healthy,
		ReadyWithin: 60 * time.Second,
		Output:      os.Stderr,
	})

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	supervised := make(chan error, 1)
	go func() { supervised <- supervisor.Run(ctx) }()

	served := make(chan error, 1)
	server := api.New(client, supervisor, Version, coreURL)
	go func() { served <- server.Serve(ctx, cfg.Listen) }()

	fmt.Fprintf(os.Stderr, "sourceant-agent %s listening on %s, core on %s\n", Version, cfg.Listen, coreURL)

	// Whichever half stops first ends the agent: an agent serving without a
	// core answers nothing, and a core nobody serves is not reachable.
	select {
	case err := <-supervised:
		return err
	case err := <-served:
		return err
	}
}
