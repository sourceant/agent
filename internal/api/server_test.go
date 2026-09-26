package api

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/sourceant/agent/internal/core"
)

type stubReader struct {
	up              bool
	repositories    []core.Repository
	graph           core.Graph
	attention       core.Attention
	indexed         []core.Indexed
	knowledge       core.KnowledgePage
	err             error
	askedFor        string
	askedOptions    core.GraphOptions
	askedEverything bool
	askedUpdate     bool
	askedDryRun     bool
	askedUseModel   bool
	seeded          core.Seeded
	skills          core.SkillPage
	oneSkill        core.Skill
	reviewed        core.Review
	readings        []core.Reading
	asked           core.Ask
	stated          core.Stated
	settings        []core.Setting
	offered         []core.Offering
	usable          core.Usable
	askedModel      string
	askedKey        string
	setKey          string
	setValue        any
	registered      core.Repository
	recorded        core.Knowledge
	forgot          string
}

func (s *stubReader) Healthy(context.Context) bool { return s.up }

func (s *stubReader) Repositories(context.Context) ([]core.Repository, error) {
	return s.repositories, s.err
}

func (s *stubReader) Graph(_ context.Context, repository string, opts core.GraphOptions) (core.Graph, error) {
	s.askedFor = repository
	s.askedOptions = opts
	return s.graph, s.err
}

func (s *stubReader) Attention(_ context.Context, repository string) (core.Attention, error) {
	s.askedFor = repository
	return s.attention, s.err
}

func (s *stubReader) Register(_ context.Context, path, name string) (core.Repository, error) {
	s.registered = core.Repository{Name: name, Path: path}
	return s.registered, s.err
}

func (s *stubReader) Forget(_ context.Context, path string) error {
	s.forgot = path
	return s.err
}

func (s *stubReader) Index(_ context.Context, repository string, everything, update bool) ([]core.Indexed, error) {
	s.askedFor = repository
	s.askedEverything = everything
	s.askedUpdate = update
	return s.indexed, s.err
}

func (s *stubReader) Knowledge(_ context.Context, repository string, limit, offset int) (core.KnowledgePage, error) {
	s.askedFor = repository
	return s.knowledge, s.err
}

func (s *stubReader) RecordKnowledge(_ context.Context, repository string, item core.Knowledge) (core.Knowledge, error) {
	s.askedFor = repository
	s.recorded = item
	return item, s.err
}

func (s *stubReader) ForgetKnowledge(_ context.Context, repository, id string) error {
	s.askedFor = repository
	s.forgot = id
	return s.err
}

func (s *stubReader) Initialize(_ context.Context, repository string, dryRun, useModel bool) (core.Seeded, error) {
	s.askedFor = repository
	s.askedDryRun = dryRun
	s.askedUseModel = useModel
	return s.seeded, s.err
}

func (s *stubReader) Skills(_ context.Context, repository string) (core.SkillPage, error) {
	s.askedFor = repository
	return s.skills, s.err
}

func (s *stubReader) Skill(_ context.Context, id, repository string) (core.Skill, error) {
	s.askedFor = repository
	s.forgot = id
	return s.oneSkill, s.err
}

func (s *stubReader) ResetSetting(_ context.Context, key string) (core.Setting, error) {
	s.setKey = key
	return core.Setting{Key: key}, s.err
}

func (s *stubReader) RecordSkill(_ context.Context, stated core.Stated) (core.Skill, error) {
	s.stated = stated
	return core.Skill{ID: stated.ID, Name: stated.Name}, s.err
}

func (s *stubReader) ForgetSkill(_ context.Context, repository, scope, id string) error {
	s.askedFor = repository
	s.forgot = id
	return s.err
}

func (s *stubReader) Review(_ context.Context, ask core.Ask) (core.Reading, error) {
	s.asked = ask
	s.askedFor = ask.Repository
	return core.Reading{ID: "one", Repository: ask.Repository, Status: "running"}, s.err
}

func (s *stubReader) Reviewed(_ context.Context, id string) (core.Reading, error) {
	s.forgot = id
	return core.Reading{ID: id, Status: "done", Review: s.reviewed}, s.err
}

func (s *stubReader) Reviews(_ context.Context, repository string) ([]core.Reading, error) {
	s.askedFor = repository
	return s.readings, s.err
}

func (s *stubReader) Settings(context.Context) ([]core.Setting, error) {
	return s.settings, s.err
}

func (s *stubReader) Models(context.Context) ([]core.Offering, error) {
	return s.offered, s.err
}

func (s *stubReader) CheckModel(_ context.Context, model, key, _ string) (core.Usable, error) {
	s.askedModel, s.askedKey = model, key
	return s.usable, s.err
}

func (s *stubReader) SetSetting(_ context.Context, key string, value any) (core.Setting, error) {
	s.setKey, s.setValue = key, value
	return core.Setting{Key: key}, s.err
}

type stubSupervisor struct {
	starts int
	exit   error
}

func (s stubSupervisor) Starts() int     { return s.starts }
func (s stubSupervisor) LastExit() error { return s.exit }

func call(t *testing.T, server *Server, target string) *httptest.ResponseRecorder {
	t.Helper()
	recorder := httptest.NewRecorder()
	server.Handler().ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, target, nil))
	return recorder
}

func TestHealthReportsWhetherTheCoreIsAnswering(t *testing.T) {
	server := New(&stubReader{up: true}, stubSupervisor{starts: 2}, "1.2.3", "http://127.0.0.1:8931")

	response := call(t, server, "/health")

	var status Status
	decode(t, response, &status)
	if !status.CoreUp {
		t.Error("reported a core that answered as down")
	}
	if status.CoreStarts != 2 {
		t.Errorf("got %d starts, want 2", status.CoreStarts)
	}
	if status.Version != "1.2.3" || status.CoreURL != "http://127.0.0.1:8931" {
		t.Errorf("got version %q at %q, want what the agent was built with", status.Version, status.CoreURL)
	}
}

func TestHealthCarriesWhyTheCoreLastDied(t *testing.T) {
	server := New(&stubReader{}, stubSupervisor{starts: 9, exit: errors.New("exit status 1")}, "dev", "")

	response := call(t, server, "/health")

	var status Status
	decode(t, response, &status)
	if status.LastExit != "exit status 1" {
		t.Errorf("got last exit %q, want what the process said", status.LastExit)
	}
}

func TestAnEmptyRegistryIsAnEmptyListRatherThanNull(t *testing.T) {
	server := New(&stubReader{repositories: nil}, stubSupervisor{}, "dev", "")

	response := call(t, server, "/api/repositories")

	if got := response.Body.String(); got != "[]\n" {
		t.Errorf("got %q, want an empty list", got)
	}
}

func TestGraphPassesOnWhatNarrowsADrawing(t *testing.T) {
	reader := &stubReader{}
	server := New(reader, stubSupervisor{}, "dev", "")

	call(t, server, "/api/graph?repository=acme/billing&path_prefix=app/&include_tests=true&node_limit=200")

	if reader.askedFor != "acme/billing" {
		t.Errorf("asked for %q, want acme/billing", reader.askedFor)
	}
	want := core.GraphOptions{PathPrefix: "app/", IncludeTests: true, NodeLimit: 200}
	if reader.askedOptions != want {
		t.Errorf("asked with %+v, want %+v", reader.askedOptions, want)
	}
}

func TestGraphRefusesToGuessWhichRepositoryIsMeant(t *testing.T) {
	server := New(&stubReader{}, stubSupervisor{}, "dev", "")

	response := call(t, server, "/api/graph")

	if response.Code != http.StatusBadRequest {
		t.Errorf("got %d, want 400", response.Code)
	}
}

func TestAnUnregisteredRepositoryStaysA404(t *testing.T) {
	reader := &stubReader{err: &core.Error{
		StatusCode: http.StatusNotFound,
		Detail:     "acme/nope is not registered on this machine",
	}}
	server := New(reader, stubSupervisor{}, "dev", "")

	response := call(t, server, "/api/graph?repository=acme/nope")

	if response.Code != http.StatusNotFound {
		t.Errorf("got %d, want the core's own 404", response.Code)
	}
	var body problem
	decode(t, response, &body)
	if body.Error != "acme/nope is not registered on this machine" {
		t.Errorf("got %q, want the reason the core gave", body.Error)
	}
}

func TestACoreThatCannotBeReachedIsNotTheAgentBreaking(t *testing.T) {
	server := New(&stubReader{err: errors.New("connection refused")}, stubSupervisor{}, "dev", "")

	response := call(t, server, "/api/repositories")

	if response.Code != http.StatusBadGateway {
		t.Errorf("got %d, want 502", response.Code)
	}
}

func decode(t *testing.T, response *httptest.ResponseRecorder, into any) {
	t.Helper()
	if err := json.Unmarshal(response.Body.Bytes(), into); err != nil {
		t.Fatalf("decoding %q: %v", response.Body.String(), err)
	}
}
