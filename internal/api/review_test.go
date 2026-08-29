package api

import (
	"encoding/json"
	"net/http"
	"testing"

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

// The agent does not hold a review. It asks the core, which keeps it, so a
// link to one still opens after this process has restarted.
func TestAskingForAReviewAnswersWithWhereToFindIt(t *testing.T) {
	reader := &stubReader{}
	server := New(reader, stubSupervisor{}, "dev", "")

	response := body(t, server, http.MethodPost, "/api/reviews",
		`{"repository":"acme/billing","against":"dev","title":"Edit the migration","use_model":true}`)

	if response.Code != http.StatusAccepted {
		t.Fatalf("got %d, want 202", response.Code)
	}
	var started core.Reading
	if err := json.Unmarshal(response.Body.Bytes(), &started); err != nil {
		t.Fatalf("could not read the answer: %v", err)
	}
	if started.ID == "" {
		t.Error("answered without a name, so nobody could come back for it")
	}
	if reader.asked.Against != "dev" || !reader.asked.UseModel {
		t.Errorf("asked %+v, want what was sent", reader.asked)
	}
}

func TestOneReviewComesBackByName(t *testing.T) {
	reader := &stubReader{reviewed: core.Review{
		Ready:    false,
		Verdicts: []core.Verdict{{Skill: "migrations", Passed: false}},
	}}
	server := New(reader, stubSupervisor{}, "dev", "")

	response := call(t, server, "/api/reviews/abc123")

	if response.Code != http.StatusOK {
		t.Fatalf("got %d, want 200", response.Code)
	}
	var found core.Reading
	if err := json.Unmarshal(response.Body.Bytes(), &found); err != nil {
		t.Fatalf("could not read the answer: %v", err)
	}
	if reader.forgot != "abc123" {
		t.Errorf("asked for %q, want the review named", reader.forgot)
	}
	if len(found.Review.Verdicts) != 1 {
		t.Errorf("got %d verdicts, want the one it made", len(found.Review.Verdicts))
	}
}

func TestTheLastFewComeBackAsAList(t *testing.T) {
	server := New(&stubReader{}, stubSupervisor{}, "dev", "")

	response := call(t, server, "/api/reviews")

	var found []core.Reading
	if err := json.Unmarshal(response.Body.Bytes(), &found); err != nil {
		t.Fatalf("could not read the answer: %v", err)
	}
	if found == nil {
		t.Error("answered null, want an empty list a screen can draw")
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

	response := call(t, server, "/api/reviews/abc123")

	var found core.Reading
	if err := json.Unmarshal(response.Body.Bytes(), &found); err != nil {
		t.Fatalf("could not read the answer: %v", err)
	}
	if found.Review.Changed == nil || found.Review.Skills == nil || found.Review.Verdicts == nil {
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
