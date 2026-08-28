// Package config reads what the agent needs to know before it starts anything.
package config

import (
	"fmt"
	"os"
	"strconv"
)

const (
	EnvListen = "SOURCEANT_AGENT_LISTEN"
	EnvCore   = "SOURCEANT_CORE"
	EnvPort   = "SOURCEANT_CORE_PORT"
)

// DefaultListen is loopback on purpose. The agent reads a person's working
// tree; the machine it runs on is the only audience it has.
const DefaultListen = "127.0.0.1:8930"

// DefaultCore is the Python entry point, found on PATH.
const DefaultCore = "sourceant"

// Config is everything the agent takes from its environment.
type Config struct {
	// Listen is where the agent answers.
	Listen string
	// Core is the Python executable to supervise.
	Core string
	// CoreWasChosen says somebody named that executable, rather than it being
	// the fallback. An explicit choice outranks whatever was installed.
	CoreWasChosen bool
	// CorePort is the port to start it on, zero to pick a free one.
	CorePort int
}

// FromEnvironment reads the configuration, filling in what was not set.
func FromEnvironment() (Config, error) {
	chosen := os.Getenv(EnvCore)
	cfg := Config{
		Listen:        valueOr(EnvListen, DefaultListen),
		Core:          valueOr(EnvCore, DefaultCore),
		CoreWasChosen: chosen != "",
	}
	if raw := os.Getenv(EnvPort); raw != "" {
		port, err := strconv.Atoi(raw)
		if err != nil || port < 1 || port > 65535 {
			return cfg, fmt.Errorf("%s is %q, which is not a port", EnvPort, raw)
		}
		cfg.CorePort = port
	}
	return cfg, nil
}

func valueOr(name, fallback string) string {
	if value := os.Getenv(name); value != "" {
		return value
	}
	return fallback
}
