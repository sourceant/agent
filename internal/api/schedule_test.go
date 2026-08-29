package api

import (
	"testing"
	"time"

	"github.com/sourceant/agent/internal/core"
)

func setting(key string, value any) core.Setting {
	return core.Setting{Key: key, Value: value}
}

func TestSomethingTurnedOffNeverRuns(t *testing.T) {
	if due(0, time.Time{}) {
		t.Error("ran something set to every zero minutes")
	}
}

func TestSomethingNeverRunIsOwedARun(t *testing.T) {
	if !due(time.Hour, time.Time{}) {
		t.Error("waited an hour before the first run")
	}
}

func TestSomethingJustRunIsNotOwedAnother(t *testing.T) {
	if due(time.Hour, time.Now()) {
		t.Error("ran again immediately")
	}
}

// Somebody typing 1 by accident should not have their laptop reading every
// repository they own once a minute.
func TestNothingRunsSoonerThanIsSensible(t *testing.T) {
	if due(time.Minute, time.Now().Add(-2*time.Minute)) {
		t.Error("honoured an interval shorter than anything is worth")
	}
	if !due(time.Minute, time.Now().Add(-10*time.Minute)) {
		t.Error("never ran at all")
	}
}

func TestHowOftenIsReadFromWhatTheCoreDeclares(t *testing.T) {
	settings := []core.Setting{setting("index.every", float64(30))}

	if minutes(settings, "index.every") != 30*time.Minute {
		t.Errorf("got %s, want 30m", minutes(settings, "index.every"))
	}
	if minutes(settings, "nothing.declared") != 0 {
		t.Error("invented a schedule for a setting nobody declared")
	}
}

func TestAModelIsOnlyAskedWhereThereIsOne(t *testing.T) {
	yes, no := true, false

	named := []core.Setting{
		setting("model.name", "gemini/gemini-2.5-flash"),
		{Key: "model.api_key", IsSet: &yes},
	}
	unkeyed := []core.Setting{
		setting("model.name", "gemini/gemini-2.5-flash"),
		{Key: "model.api_key", IsSet: &no},
	}

	if !configured(named) {
		t.Error("would not ask a model that is configured")
	}
	if configured(unkeyed) {
		t.Error("would ask a model with no key, which is refused every time")
	}
	if configured(nil) {
		t.Error("would ask a model nobody chose")
	}
}
