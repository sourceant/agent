package supervise

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"sync/atomic"
	"testing"
	"time"
)

// script writes an executable shell script and returns the command to run it.
func script(t *testing.T, body string) (string, []string) {
	t.Helper()
	path := filepath.Join(t.TempDir(), "process.sh")
	if err := os.WriteFile(path, []byte("#!/bin/sh\n"+body), 0o755); err != nil {
		t.Fatalf("writing script: %v", err)
	}
	return "/bin/sh", []string{path}
}

func TestItRestartsAProcessThatDies(t *testing.T) {
	name, args := script(t, "exit 1\n")
	supervisor := New(Options{
		Name:    name,
		Args:    args,
		Backoff: Backoff{First: time.Millisecond, Max: 2 * time.Millisecond},
	})

	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() { done <- supervisor.Run(ctx) }()

	waitFor(t, func() bool { return supervisor.Starts() >= 3 })
	cancel()

	if err := <-done; err != nil {
		t.Fatalf("running: %v", err)
	}
	if supervisor.LastExit() == nil {
		t.Error("a process that exited 1 was recorded as exiting cleanly")
	}
}

func TestItLeavesNothingRunningWhenItIsToldToStop(t *testing.T) {
	marker := filepath.Join(t.TempDir(), "alive")
	name, args := script(t, "touch "+marker+"\nwhile true; do sleep 0.05; done\n")
	supervisor := New(Options{Name: name, Args: args})

	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() { done <- supervisor.Run(ctx) }()

	waitFor(t, func() bool {
		_, err := os.Stat(marker)
		return err == nil
	})
	cancel()

	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("running: %v", err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("cancelling left the process running")
	}
}

// A core that started a worker leaves it holding the port unless the whole
// group is stopped, so this drives a process that starts a child of its own.
func TestItLeavesNoGrandchildRunningEither(t *testing.T) {
	dir := t.TempDir()
	marker := filepath.Join(dir, "child-alive")
	child := filepath.Join(dir, "child.sh")
	if err := os.WriteFile(child, []byte(
		"#!/bin/sh\nwhile true; do touch "+marker+"; sleep 0.02; done\n"), 0o755); err != nil {
		t.Fatalf("writing child script: %v", err)
	}
	name, args := script(t, child+" &\nwhile true; do sleep 0.05; done\n")
	supervisor := New(Options{Name: name, Args: args, StopWithin: 200 * time.Millisecond})

	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() { done <- supervisor.Run(ctx) }()

	waitFor(t, func() bool {
		_, err := os.Stat(marker)
		return err == nil
	})
	cancel()
	<-done

	// The child touches the marker every 20ms while it lives. Remove it, and if
	// it comes back the child outlived the process that started it.
	if err := os.Remove(marker); err != nil {
		t.Fatalf("removing marker: %v", err)
	}
	time.Sleep(300 * time.Millisecond)
	if _, err := os.Stat(marker); err == nil {
		t.Error("a grandchild was still running after the supervisor stopped")
	}
}

func TestItWaitsForTheProcessToAnswerBeforeCallingItStarted(t *testing.T) {
	name, args := script(t, "while true; do sleep 0.05; done\n")
	var polls atomic.Int32
	supervisor := New(Options{
		Name: name,
		Args: args,
		Ready: func(context.Context) bool {
			return polls.Add(1) >= 3
		},
		PollEvery:   time.Millisecond,
		ReadyWithin: 5 * time.Second,
	})

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go func() { _ = supervisor.Run(ctx) }()

	waitFor(t, func() bool { return polls.Load() >= 3 })
	if supervisor.Starts() != 1 {
		t.Errorf("started %d times while waiting to be ready, want 1", supervisor.Starts())
	}
}

func TestAProcessThatNeverAnswersIsReportedRatherThanRestartedForever(t *testing.T) {
	name, args := script(t, "while true; do sleep 0.05; done\n")
	supervisor := New(Options{
		Name:        name,
		Args:        args,
		Ready:       func(context.Context) bool { return false },
		PollEvery:   time.Millisecond,
		ReadyWithin: 50 * time.Millisecond,
	})

	err := supervisor.Run(context.Background())

	if !errors.Is(err, ErrNotReady) {
		t.Fatalf("got %v, want ErrNotReady", err)
	}
}

func TestFreePortGivesAPortNobodyIsUsing(t *testing.T) {
	port, err := FreePort()
	if err != nil {
		t.Fatalf("asking for a port: %v", err)
	}
	if port <= 0 || port > 65535 {
		t.Errorf("got port %d, want a usable one", port)
	}
}

func waitFor(t *testing.T, condition func() bool) {
	t.Helper()
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		if condition() {
			return
		}
		time.Sleep(2 * time.Millisecond)
	}
	t.Fatal("condition never held")
}
