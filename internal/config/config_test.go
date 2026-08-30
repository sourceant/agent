package config

import "testing"

func TestItRunsWithNothingSet(t *testing.T) {
	t.Setenv(EnvListen, "")
	t.Setenv(EnvCore, "")
	t.Setenv(EnvPort, "")

	cfg, err := FromEnvironment()
	if err != nil {
		t.Fatalf("reading configuration: %v", err)
	}

	if cfg.Listen != DefaultListen {
		t.Errorf("got %q, want %q", cfg.Listen, DefaultListen)
	}
	if cfg.Core != DefaultCore {
		t.Errorf("got %q, want %q", cfg.Core, DefaultCore)
	}
	if cfg.CorePort != 0 {
		t.Errorf("got port %d, want one chosen at start", cfg.CorePort)
	}
}

func TestItTakesWhatWasSet(t *testing.T) {
	t.Setenv(EnvListen, "127.0.0.1:9999")
	t.Setenv(EnvCore, "/opt/sourceant/bin/sourceant")
	t.Setenv(EnvPort, "8123")

	cfg, err := FromEnvironment()
	if err != nil {
		t.Fatalf("reading configuration: %v", err)
	}

	if cfg.Listen != "127.0.0.1:9999" || cfg.Core != "/opt/sourceant/bin/sourceant" || cfg.CorePort != 8123 {
		t.Errorf("got %+v, want what the environment said", cfg)
	}
}

func TestAPortThatIsNotAPortIsRefusedAtStart(t *testing.T) {
	t.Setenv(EnvPort, "eighty")

	if _, err := FromEnvironment(); err == nil {
		t.Fatal("accepted a port that is not a number")
	}
}

func TestAPortOutsideTheRangeIsRefusedAtStart(t *testing.T) {
	t.Setenv(EnvPort, "70000")

	if _, err := FromEnvironment(); err == nil {
		t.Fatal("accepted a port no socket can bind")
	}
}
