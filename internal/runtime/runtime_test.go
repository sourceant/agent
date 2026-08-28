package runtime

import (
	"errors"
	"path/filepath"
	"slices"
	"strings"
	"testing"
)

func TestNothingInstalledIsSaidPlainly(t *testing.T) {
	_, err := Load(filepath.Join(t.TempDir(), "config.json"))

	if !errors.Is(err, ErrNotInstalled) {
		t.Fatalf("got %v, want ErrNotInstalled", err)
	}
}

func TestWhatTheInstallerWroteIsWhatTheAgentReads(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.json")
	written := Config{Core: Core{
		Runtime: Docker,
		Image:   "ghcr.io/sourceant/sourceant:v1",
		DataDir: "/home/someone/.local/share/sourceant",
	}}

	if err := Save(path, written); err != nil {
		t.Fatalf("saving: %v", err)
	}
	read, err := Load(path)
	if err != nil {
		t.Fatalf("loading: %v", err)
	}

	if read != written {
		t.Errorf("got %+v, want %+v", read, written)
	}
}

func TestARuntimeThatIsNeitherIsRefusedWhenItIsRead(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.json")
	if err := Save(path, Config{Core: Core{Runtime: "podman"}}); err != nil {
		t.Fatalf("saving: %v", err)
	}

	if _, err := Load(path); err == nil {
		t.Fatal("accepted a runtime that is neither python nor docker")
	}
}

func TestThePythonRuntimeBindsLoopbackDirectly(t *testing.T) {
	core := Core{Runtime: Python, Command: "/opt/sourceant/bin/sourceant"}

	name, args, err := core.Serve(8931)
	if err != nil {
		t.Fatalf("building the command: %v", err)
	}

	if name != "/opt/sourceant/bin/sourceant" {
		t.Errorf("got %q, want the installed command", name)
	}
	want := []string{"serve", "--host", "127.0.0.1", "--port", "8931"}
	if !slices.Equal(args, want) {
		t.Errorf("got %v, want %v", args, want)
	}
}

// A container binding loopback binds its own, which nothing can reach, so it
// has to bind every interface inside and be published to loopback outside.
func TestTheDockerRuntimeBindsInsideAndPublishesOutside(t *testing.T) {
	core := Core{Runtime: Docker, Image: "ghcr.io/sourceant/sourceant:v1", DataDir: "/data/here", User: "501:20"}

	name, args, err := core.Serve(8931)
	if err != nil {
		t.Fatalf("building the command: %v", err)
	}

	if name != "docker" {
		t.Errorf("got %q, want docker", name)
	}
	line := strings.Join(args, " ")
	for _, want := range []string{
		"-p 127.0.0.1:8931:8931",
		"--host 0.0.0.0 --port 8931",
		"-v /data/here:/data",
		"-e SOURCEANT_HOME=/data",
		"--user 501:20",
		"--entrypoint ./sourceant",
		"ghcr.io/sourceant/sourceant:v1 serve",
	} {
		if !strings.Contains(line, want) {
			t.Errorf("%q is missing from: docker %s", want, line)
		}
	}
}

// Every docker flag has to come before the image, or docker reads it as an
// argument to the core instead of to itself.
func TestEveryDockerFlagComesBeforeTheImage(t *testing.T) {
	core := Core{Runtime: Docker, Image: "ghcr.io/sourceant/sourceant:v1", DataDir: "/data/here", User: "501:20"}

	_, args, _ := core.Serve(8931)

	image := slices.Index(args, "ghcr.io/sourceant/sourceant:v1")
	if image == -1 {
		t.Fatal("the image is not in the command")
	}
	for _, flag := range []string{"--rm", "--name", "-p", "-v", "-e", "--user", "--entrypoint"} {
		if at := slices.Index(args, flag); at > image {
			t.Errorf("%s comes after the image, so docker would pass it to the core", flag)
		}
	}
}

func TestADataDirNobodyChoseIsNotMounted(t *testing.T) {
	core := Core{Runtime: Docker, Image: "img"}

	_, args, _ := core.Serve(8931)

	if slices.Contains(args, "-v") {
		t.Errorf("mounted something without being told where: %v", args)
	}
}
