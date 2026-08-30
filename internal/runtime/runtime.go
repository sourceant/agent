// Package runtime says how to start the Python core on this machine.
//
// There are two ways to have it, and which one a person chose is a fact about
// their machine rather than about either binary: the installer writes it down
// and the agent reads it. Without that file the agent looks for the core on
// PATH, which is what a developer working on the core itself already has.
package runtime

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
)

// Kind is how the core is installed.
type Kind string

const (
	// Python is the core as a program on this machine.
	Python Kind = "python"
	// Docker is the core as a container image.
	Docker Kind = "docker"
)

// DefaultImage is the published core. A tag is pinned at install time; this is
// only the fallback for a config that names none.
const DefaultImage = "ghcr.io/sourceant/sourceant:latest"

// Core is everything needed to start the indexer.
type Core struct {
	Runtime Kind `json:"runtime"`
	// Command is the executable, for the python runtime.
	Command string `json:"command,omitempty"`
	// Image is the container, for the docker runtime.
	Image string `json:"image,omitempty"`
	// DataDir is where the index lives. Both runtimes must agree on it, or
	// indexing and reading would address two different databases.
	DataDir string `json:"data_dir,omitempty"`
	// Mount is a host directory the container can see, for the docker runtime.
	//
	// The indexer reads the repository's files, so a container that cannot see
	// them indexes nothing. It is mounted at the same path it has on the host,
	// which is what lets one registry of absolute paths mean the same thing to
	// both runtimes. A repository outside it is not readable this way.
	Mount string `json:"mount,omitempty"`
	// UIURL is where the agent serves the screen, so the core can hand out a
	// link to a review rather than a path.
	UIURL string `json:"-"`
	// User is the uid:gid a container runs as, for the docker runtime.
	//
	// The image has a user of its own, and where that user's id differs from
	// the person's, everything the container writes into the mounted index
	// belongs to somebody who does not exist on this machine. The installer
	// records who is installing so the container writes as them.
	User string `json:"user,omitempty"`
}

// Config is what the installer wrote.
type Config struct {
	Core Core `json:"core"`
}

// Home is where SourceAnt keeps what it installed.
func Home() string {
	if override := os.Getenv("SOURCEANT_INSTALL_HOME"); override != "" {
		return override
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return ".sourceant"
	}
	return filepath.Join(home, ".sourceant")
}

// ConfigPath is the file the installer writes and the agent reads.
func ConfigPath() string { return filepath.Join(Home(), "config.json") }

// ErrNotInstalled says nothing has been installed here yet.
var ErrNotInstalled = errors.New("no runtime is installed")

// Load reads the installed runtime.
func Load(path string) (Config, error) {
	data, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return Config{}, ErrNotInstalled
	}
	if err != nil {
		return Config{}, err
	}
	var config Config
	if err := json.Unmarshal(data, &config); err != nil {
		return Config{}, fmt.Errorf("%s is not readable as a runtime: %w", path, err)
	}
	if config.Core.Runtime != Python && config.Core.Runtime != Docker {
		return Config{}, fmt.Errorf("%s names runtime %q, which is neither python nor docker", path, config.Core.Runtime)
	}
	return config, nil
}

// Save writes the runtime for the agent to read.
func Save(path string, config Config) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, append(data, '\n'), 0o644)
}

// Serve is the command that starts the core listening on port.
//
// The two runtimes need different addresses for the same thing. A program on
// this machine binds loopback and is reached there. A container binding
// loopback would bind the container's own, reachable by nothing, so it binds
// every interface inside and is published to loopback outside.
func (c Core) Serve(port int) (string, []string, error) {
	number := strconv.Itoa(port)
	switch c.Runtime {
	case Python:
		if c.Command == "" {
			return "", nil, errors.New("the python runtime names no command")
		}
		return c.Command, []string{"serve", "--host", "127.0.0.1", "--port", number}, nil
	case Docker:
		image := c.Image
		if image == "" {
			image = DefaultImage
		}
		args := []string{
			"run", "--rm",
			"--name", "sourceant-core-" + number,
			"-p", "127.0.0.1:" + number + ":" + number,
		}
		if c.DataDir != "" {
			args = append(args, "-v", c.DataDir+":/data", "-e", "SOURCEANT_HOME=/data")
		}
		if c.UIURL != "" {
			// So a review asked for over MCP can answer with a link somebody
			// can click, rather than a path they have to assemble.
			args = append(args, "-e", "SOURCEANT_UI_URL="+c.UIURL)
			// And a second address for reaching back. The clickable one is
			// loopback, which inside a container is the container, so handing
			// work to the agent needs the host's address instead.
			args = append(args,
				"--add-host", "host.docker.internal:host-gateway",
				"-e", "SOURCEANT_AGENT_URL="+throughTheHost(c.UIURL),
			)
		}
		if c.Mount != "" {
			args = append(args, "-v", c.Mount+":"+c.Mount)
			// The image has a home of its own, and nothing a person taught
			// their coding agent is in it. What the person keeps in theirs is
			// only readable if the container is told where theirs is.
			args = append(args, "-e", "SOURCEANT_MACHINE_HOME="+c.Mount)
		}
		if c.User != "" {
			args = append(args, "--user", c.User)
		}
		// The image starts a production server by default, so the entry point
		// is replaced with the command line the agent actually wants.
		args = append(args, "--entrypoint", "./sourceant", image,
			"serve", "--host", "0.0.0.0", "--port", number)
		return "docker", args, nil
	default:
		return "", nil, fmt.Errorf("runtime %q is neither python nor docker", c.Runtime)
	}
}

// Describe is one line naming what will be started.
func (c Core) Describe() string {
	switch c.Runtime {
	case Python:
		return "python · " + c.Command
	case Docker:
		image := c.Image
		if image == "" {
			image = DefaultImage
		}
		return "docker · " + image
	default:
		return string(c.Runtime)
	}
}

// throughTheHost rewrites a loopback address into one a container can reach.
//
// The agent listens on loopback, which is right: nothing else should reach it.
// A container's loopback is its own, so the same URL means two different
// machines depending on who reads it.
func throughTheHost(address string) string {
	parsed, err := url.Parse(address)
	if err != nil {
		return address
	}
	host := parsed.Hostname()
	if host != "127.0.0.1" && host != "localhost" && host != "::1" {
		return address
	}
	if port := parsed.Port(); port != "" {
		parsed.Host = "host.docker.internal:" + port
	} else {
		parsed.Host = "host.docker.internal"
	}
	return parsed.String()
}
