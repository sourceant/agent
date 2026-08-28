// Package supervise keeps the Python core running.
//
// The core is a separate process because the grammars and the graph live there.
// The agent's job is to make that process something a person never has to think
// about: start it, notice when it dies, start it again.
package supervise

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"os/exec"
	"sync"
	"time"
)

// Backoff is how long to wait before restarting, and how far that grows.
type Backoff struct {
	First time.Duration
	Max   time.Duration
}

func (b Backoff) after(previous time.Duration) time.Duration {
	if previous == 0 {
		return b.First
	}
	doubled := previous * 2
	if doubled > b.Max {
		return b.Max
	}
	return doubled
}

// Options describe the process to keep alive.
type Options struct {
	// Name and Args are the command. Name is looked up on PATH.
	Name string
	Args []string
	// Dir is where it runs, empty for the agent's own directory.
	Dir string
	// Env is the whole environment, empty to inherit the agent's.
	Env []string
	// Ready reports whether the process is up and answering. It is polled
	// after each start until it says yes or ReadyWithin passes.
	Ready func(context.Context) bool
	// ReadyWithin bounds that wait, and PollEvery is how often it is asked.
	ReadyWithin time.Duration
	PollEvery   time.Duration
	// StopWithin is how long a process gets to stop when asked, before it is
	// killed.
	StopWithin time.Duration
	// Backoff paces restarts.
	Backoff Backoff
	// Output receives the process's stdout and stderr, nil to discard.
	Output io.Writer
}

func (o Options) withDefaults() Options {
	if o.ReadyWithin == 0 {
		o.ReadyWithin = 30 * time.Second
	}
	if o.PollEvery == 0 {
		o.PollEvery = 100 * time.Millisecond
	}
	if o.StopWithin == 0 {
		o.StopWithin = 5 * time.Second
	}
	if o.Backoff.First == 0 {
		o.Backoff.First = 250 * time.Millisecond
	}
	if o.Backoff.Max == 0 {
		o.Backoff.Max = 30 * time.Second
	}
	return o
}

// ErrNotReady says the process started but never began answering.
var ErrNotReady = errors.New("the process started but never became ready")

// process is one launch, and the single place its exit is waited for.
//
// exec.Cmd is not safe to Wait on from one goroutine while another reads
// ProcessState, so exactly one goroutine calls Wait and everybody else learns
// what happened by waiting on done.
type process struct {
	cmd  *exec.Cmd
	err  error
	done chan struct{}
}

// wait is the only caller of Wait. Writing err before closing done is what lets
// exit read it without a lock.
func (p *process) wait() {
	p.err = p.cmd.Wait()
	close(p.done)
}

// exit blocks until the process has gone, and says why.
func (p *process) exit() error {
	<-p.done
	return p.err
}

// Supervisor runs one process and restarts it for as long as it is asked to.
type Supervisor struct {
	options Options

	mu       sync.Mutex
	starts   int
	lastExit error
}

// New builds a supervisor for the process described by options.
func New(options Options) *Supervisor {
	return &Supervisor{options: options.withDefaults()}
}

// Starts is how many times the process has been launched, restarts included.
func (s *Supervisor) Starts() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.starts
}

// LastExit is why the process last stopped, nil if it never has.
func (s *Supervisor) LastExit() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.lastExit
}

// Run starts the process and keeps it running until ctx is cancelled.
//
// It returns once the process is stopped and no further restart is wanted, so a
// caller runs it in its own goroutine. A cancelled context is not an error: it
// is how a caller says stop.
func (s *Supervisor) Run(ctx context.Context) error {
	var wait time.Duration
	for {
		if ctx.Err() != nil {
			return nil
		}

		started, err := s.start(ctx)
		if err != nil {
			return err
		}

		if s.options.Ready != nil && !s.waitReady(ctx, started) {
			s.terminate(started)
			s.record(started.exit())
			if ctx.Err() != nil {
				return nil
			}
			return fmt.Errorf("%w within %s", ErrNotReady, s.options.ReadyWithin)
		}

		s.record(started.exit())
		if ctx.Err() != nil {
			return nil
		}

		wait = s.options.Backoff.after(wait)
		select {
		case <-ctx.Done():
			return nil
		case <-time.After(wait):
		}
	}
}

func (s *Supervisor) start(ctx context.Context) (*process, error) {
	command := exec.Command(s.options.Name, s.options.Args...)
	command.Dir = s.options.Dir
	command.Env = s.options.Env
	command.Stdout = s.options.Output
	command.Stderr = s.options.Output
	isolate(command)

	if err := command.Start(); err != nil {
		return nil, fmt.Errorf("starting %s: %w", s.options.Name, err)
	}

	started := &process{cmd: command, done: make(chan struct{})}
	go started.wait()

	s.mu.Lock()
	s.starts++
	s.mu.Unlock()

	// Stopping the process when the context ends is what makes the wait
	// return, so a cancelled Run does not leave the core running behind it.
	go func() {
		select {
		case <-ctx.Done():
			s.terminate(started)
		case <-started.done:
		}
	}()

	return started, nil
}

// waitReady polls until the process answers, it exits, or the deadline passes.
func (s *Supervisor) waitReady(ctx context.Context, started *process) bool {
	deadline := time.Now().Add(s.options.ReadyWithin)
	ticker := time.NewTicker(s.options.PollEvery)
	defer ticker.Stop()

	for {
		if s.options.Ready(ctx) {
			return true
		}
		if time.Now().After(deadline) {
			return false
		}
		select {
		case <-started.done:
			return false
		case <-ctx.Done():
			return false
		case <-ticker.C:
		}
	}
}

// terminate asks the process to stop, then insists.
//
// A core killed outright loses whatever it was writing, so it is asked first
// and only killed if it does not go. Both signals address the process group,
// because a core that started a worker leaves it holding the port otherwise.
func (s *Supervisor) terminate(started *process) {
	if started == nil || started.cmd.Process == nil {
		return
	}
	if err := askToStop(started.cmd); err != nil {
		return
	}
	select {
	case <-started.done:
	case <-time.After(s.options.StopWithin):
		_ = forceStop(started.cmd)
	}
}

func (s *Supervisor) record(exit error) {
	s.mu.Lock()
	s.lastExit = exit
	s.mu.Unlock()
}

// FreePort asks the operating system for a port nobody is using.
//
// It is racy by nature: the port is free when asked and could be taken before
// it is bound. Nothing better exists without binding it here and handing the
// listener over, which a child process cannot accept.
func FreePort() (int, error) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return 0, err
	}
	defer func() { _ = listener.Close() }()
	return listener.Addr().(*net.TCPAddr).Port, nil
}
