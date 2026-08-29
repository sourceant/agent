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
	"github.com/sourceant/agent/internal/runtime"
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

	installed := resolveCore(cfg)
	// So a review asked for over MCP answers with a link somebody can click
	// rather than a path they have to assemble. The agent serves the screen, so
	// it is the only thing that knows this.
	installed.UIURL = "http://" + cfg.Listen
	name, args, err := installed.Serve(port)
	if err != nil {
		return err
	}

	supervisor := supervise.New(supervise.Options{
		Name:        name,
		Args:        args,
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

	// A repository read once answers about last month, so it is read again on
	// whatever schedule somebody set. This is the process that is always up,
	// which is what makes it the one to do it.
	go server.Keep(ctx)

	fmt.Fprintf(os.Stderr, "sourceant-agent %s listening on %s, core on %s (%s)\n",
		Version, cfg.Listen, coreURL, installed.Describe())

	// Whichever half stops first ends the agent: an agent serving without a
	// core answers nothing, and a core nobody serves is not reachable.
	select {
	case err := <-supervised:
		return err
	case err := <-served:
		return err
	}
}

// resolveCore decides which core to start, most specific first.
//
// An explicit command wins, because somebody naming one means it. Then what
// the installer wrote. Then the core on PATH, which is what a person working
// on the core itself already has and what makes the agent runnable before
// anything has been installed at all.
func resolveCore(cfg config.Config) runtime.Core {
	if cfg.CoreWasChosen {
		return runtime.Core{Runtime: runtime.Python, Command: cfg.Core}
	}
	if installed, err := runtime.Load(runtime.ConfigPath()); err == nil {
		return installed.Core
	}
	return runtime.Core{Runtime: runtime.Python, Command: cfg.Core}
}
