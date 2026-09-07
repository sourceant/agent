package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"syscall"
	"testing"
	"time"
)

func TestAgentProcess(t *testing.T) {
	switch os.Getenv("SOURCEANT_TEST_PROCESS") {
	case "agent":
		if err := run(); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		os.Exit(0)
	case "core":
		port := os.Args[len(os.Args)-1]
		stopped := make(chan os.Signal, 1)
		signal.Notify(stopped, syscall.SIGTERM)
		go func() {
			<-stopped
			time.Sleep(200 * time.Millisecond)
			if err := os.WriteFile(os.Getenv("SOURCEANT_TEST_STOPPED"), []byte("stopped"), 0600); err != nil {
				os.Exit(1)
			}
			os.Exit(0)
		}()
		_ = http.ListenAndServe("127.0.0.1:"+port, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(http.StatusOK) }))
		os.Exit(1)
	}
}

func TestShutdownWaitsForCore(t *testing.T) {
	for _, mode := range []string{"http", "signal"} {
		t.Run(mode, func(t *testing.T) {
			dir := t.TempDir()
			binary, err := os.Executable()
			if err != nil {
				t.Fatal(err)
			}
			wrapper := filepath.Join(dir, "core")
			if err := os.WriteFile(wrapper, []byte("#!/bin/sh\nexport SOURCEANT_TEST_PROCESS=core\nexec \"$SOURCEANT_TEST_BINARY\" -test.run=TestAgentProcess -- \"$@\"\n"), 0700); err != nil {
				t.Fatal(err)
			}
			listener, err := net.Listen("tcp", "127.0.0.1:0")
			if err != nil {
				t.Fatal(err)
			}
			address := listener.Addr().String()
			_ = listener.Close()
			stopped := filepath.Join(dir, "stopped")
			cmd := exec.Command(binary, "-test.run=TestAgentProcess")
			cmd.Env = append(os.Environ(), "SOURCEANT_TEST_PROCESS=agent", "SOURCEANT_TEST_BINARY="+binary, "SOURCEANT_TEST_STOPPED="+stopped, "SOURCEANT_CORE="+wrapper, "SOURCEANT_CORE_PORT=", "SOURCEANT_AGENT_LISTEN="+address)
			var output bytes.Buffer
			cmd.Stdout = &output
			cmd.Stderr = &output
			if err := cmd.Start(); err != nil {
				t.Fatal(err)
			}
			done := make(chan struct{})
			var waitErr error
			go func() { waitErr = cmd.Wait(); close(done) }()
			t.Cleanup(func() {
				_ = cmd.Process.Signal(syscall.SIGTERM)
				select {
				case <-done:
				case <-time.After(10 * time.Second):
					_ = cmd.Process.Kill()
				}
			})
			client := &http.Client{Timeout: 10 * time.Second}
			ready := false
			deadline := time.Now().Add(10 * time.Second)
			for time.Now().Before(deadline) {
				response, err := client.Get("http://" + address + "/health")
				if err == nil {
					var status struct {
						CoreUp bool `json:"core_up"`
					}
					decodeErr := json.NewDecoder(response.Body).Decode(&status)
					_ = response.Body.Close()
					if decodeErr == nil && status.CoreUp {
						ready = true
						break
					}
				}
				time.Sleep(20 * time.Millisecond)
			}
			if !ready {
				t.Fatal("agent did not become ready")
			}
			if mode == "http" {
				request, _ := http.NewRequest(http.MethodPost, "http://"+address+"/api/stop", nil)
				request.Header.Set("X-Sourceant-Client", "cli")
				response, err := client.Do(request)
				if err != nil {
					t.Fatal(err)
				}
				_ = response.Body.Close()
				if response.StatusCode != 204 {
					t.Fatalf("stop returned %d", response.StatusCode)
				}
			} else {
				if err := cmd.Process.Signal(syscall.SIGTERM); err != nil {
					t.Fatal(err)
				}
			}
			select {
			case <-done:
				if waitErr != nil {
					t.Fatalf("%v: %s", waitErr, output.String())
				}
			case <-time.After(10 * time.Second):
				t.Fatal("agent did not stop")
			}
			if _, err := os.Stat(stopped); err != nil {
				t.Fatal("agent exited before core cleanup completed")
			}
		})
	}
}
