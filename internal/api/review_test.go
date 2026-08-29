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

func TestWorkIsReviewedAgainstWhatWasAsked(t *testing.T) {
	reader := &stubReader{reviewed: core.Review{
		Ready:    false,
		Verdicts: []core.Verdict{{Skill: "migrations", Passed: false}},
	}}
	server := New(reader, stubSupervisor{}, "dev", "")

	response := body(t, server, http.MethodPost, "/api/reviews",
		`{"repository":"acme/billing","against":"dev","title":"Edit the migration","use_model":true}`)

	if response.Code != http.StatusOK {
		t.Fatalf("got %d, want 200", response.Code)
	}
	if reader.asked.Against != "dev" || reader.asked.Title != "Edit the migration" {
		t.Errorf("asked %+v, want what was sent", reader.asked)
	}
	if !reader.asked.UseModel {
		t.Error("did not carry the ask to judge the work")
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

	response := body(t, server, http.MethodPost, "/api/reviews", `{"repository":"acme/billing"}`)

	var reviewed struct {
		Changed  []core.ChangedFile `json:"changed"`
		Skills   []core.Skill       `json:"skills"`
		Verdicts []core.Verdict     `json:"verdicts"`
	}
	if err := json.Unmarshal(response.Body.Bytes(), &reviewed); err != nil {
		t.Fatalf("could not read the answer: %v", err)
	}
	if reviewed.Changed == nil || reviewed.Skills == nil || reviewed.Verdicts == nil {
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
