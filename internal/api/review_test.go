package api

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/sourceant/agent/internal/core"
)

func TestSkillsOnHandAreListed(t *testing.T) {
	reader := &stubReader{skills: core.SkillPage{
		Skills: []core.Skill{{ID: "migrations", Name: "migrations", Origin: "repository"}},
		Total:  1,
	}}
	server := New(reader, stubSupervisor{}, "dev", "")

	response := call(t, server, "/api/skills?repository=acme/billing")

	if response.Code != http.StatusOK {
		t.Fatalf("got %d, want 200", response.Code)
	}
	if reader.askedFor != "acme/billing" {
		t.Errorf("asked for %q, want the repository named", reader.askedFor)
	}
}

func TestAMachineWithNoSkillsAnswersAnEmptyList(t *testing.T) {
	server := New(&stubReader{}, stubSupervisor{}, "dev", "")

	response := call(t, server, "/api/skills")

	var page struct {
		Skills []core.Skill `json:"skills"`
	}
	if err := json.Unmarshal(response.Body.Bytes(), &page); err != nil {
		t.Fatalf("could not read the answer: %v", err)
	}
	if page.Skills == nil {
		t.Error("answered null, want an empty list a screen can draw")
	}
}

func TestOneSkillComesBackInFull(t *testing.T) {
	reader := &stubReader{oneSkill: core.Skill{ID: "migrations", Body: "Never edit one."}}
	server := New(reader, stubSupervisor{}, "dev", "")

	response := call(t, server, "/api/skills/migrations")

	if response.Code != http.StatusOK {
		t.Fatalf("got %d, want 200", response.Code)
	}
	if reader.forgot != "migrations" {
		t.Errorf("asked for %q, want the skill named", reader.forgot)
	}
}

// A review takes tens of seconds, so the request that starts it does not wait
// for it. Anything that interrupts a connection held that long loses work that
// had already been paid for.
func waitFor(t *testing.T, server *Server, id string) job {
	t.Helper()
	for range 200 {
		response := call(t, server, "/api/reviews/"+id)
		if response.Code != http.StatusOK {
			t.Fatalf("asking how it went: got %d, want 200", response.Code)
		}
		var one job
		if err := json.Unmarshal(response.Body.Bytes(), &one); err != nil {
			t.Fatalf("could not read the answer: %v", err)
		}
		if one.Status != Running {
			return one
		}
		time.Sleep(5 * time.Millisecond)
	}
	t.Fatal("the review never finished")
	return job{}
}

func started(t *testing.T, server *Server, payload string) string {
	t.Helper()
	response := body(t, server, http.MethodPost, "/api/reviews", payload)
	if response.Code != http.StatusAccepted {
		t.Fatalf("got %d, want 202", response.Code)
	}
	var one job
	if err := json.Unmarshal(response.Body.Bytes(), &one); err != nil {
		t.Fatalf("could not read the answer: %v", err)
	}
	if one.ID == "" || one.Status != Running {
		t.Fatalf("got %+v, want a review that is running", one)
	}
	return one.ID
}

func TestWorkIsReviewedAgainstWhatWasAsked(t *testing.T) {
	reader := &stubReader{reviewed: core.Review{
		Ready:    false,
		Verdicts: []core.Verdict{{Skill: "migrations", Passed: false}},
	}}
	server := New(reader, stubSupervisor{}, "dev", "")

	id := started(t, server, `{"repository":"acme/billing","against":"dev","title":"Edit the migration","use_model":true}`)
	done := waitFor(t, server, id)

	if done.Status != Done {
		t.Fatalf("got %q: %s", done.Status, done.Error)
	}
	if reader.asked.Against != "dev" || reader.asked.Title != "Edit the migration" {
		t.Errorf("asked %+v, want what was sent", reader.asked)
	}
	if !reader.asked.UseModel {
		t.Error("did not carry the ask to judge the work")
	}
	if len(done.Review.Verdicts) != 1 {
		t.Errorf("got %d verdicts, want the one the review made", len(done.Review.Verdicts))
	}
}

// A page that was closed, or reloaded, comes back for the answer.
func TestTheAnswerIsHeldUntilSomebodyAsksForIt(t *testing.T) {
	server := New(&stubReader{reviewed: core.Review{Ready: true}}, stubSupervisor{}, "dev", "")

	id := started(t, server, `{"repository":"acme/billing"}`)
	waitFor(t, server, id)

	again := waitFor(t, server, id)

	if again.Status != Done || !again.Review.Ready {
		t.Errorf("got %+v, want the answer still there", again)
	}
}

func TestAReviewNobodyStartedIsNotInvented(t *testing.T) {
	server := New(&stubReader{}, stubSupervisor{}, "dev", "")

	if call(t, server, "/api/reviews/nothing").Code != http.StatusNotFound {
		t.Error("answered for a review that was never started")
	}
}

func TestWhatWentWrongIsKeptRatherThanLost(t *testing.T) {
	server := New(&stubReader{err: errors.New("the provider said no")}, stubSupervisor{}, "dev", "")

	done := waitFor(t, server, started(t, server, `{"repository":"acme/billing"}`))

	if done.Status != Failed {
		t.Fatalf("got %q, want failed", done.Status)
	}
	if !strings.Contains(done.Error, "the provider said no") {
		t.Errorf("got %q, want what actually went wrong", done.Error)
	}
}

func TestReviewingNowhereIsRefused(t *testing.T) {
	server := New(&stubReader{}, stubSupervisor{}, "dev", "")

	response := body(t, server, http.MethodPost, "/api/reviews", `{}`)

	if response.Code != http.StatusBadRequest {
		t.Errorf("got %d, want 400", response.Code)
	}
}

func TestAReviewWithNothingToSayStillDrawsAsLists(t *testing.T) {
	server := New(&stubReader{reviewed: core.Review{Ready: true}}, stubSupervisor{}, "dev", "")

	done := waitFor(t, server, started(t, server, `{"repository":"acme/billing"}`))

	if done.Review.Changed == nil || done.Review.Skills == nil || done.Review.Verdicts == nil {
		t.Error("answered null somewhere, want empty lists a screen can draw")
	}
}

func TestAskingAModelToInitializeIsCarriedThrough(t *testing.T) {
	reader := &stubReader{}
	server := New(reader, stubSupervisor{}, "dev", "")

	body(t, server, http.MethodPost, "/api/knowledge/initialize",
		`{"repository":"acme/billing","use_model":true}`)

	if !reader.askedUseModel {
		t.Error("did not carry the ask to use a model, so nothing would be proposed")
	}
}
