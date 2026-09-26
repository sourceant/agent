package api

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/sourceant/agent/internal/core"
)

func body(t *testing.T, server *Server, method, target, payload string) *httptest.ResponseRecorder {
	t.Helper()
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(method, target, strings.NewReader(payload))
	server.Handler().ServeHTTP(recorder, request)
	return recorder
}

func TestADirectoryCanBeCovered(t *testing.T) {
	reader := &stubReader{}
	server := New(reader, stubSupervisor{}, "dev", "")

	response := body(t, server, http.MethodPost, "/api/repositories",
		`{"path":"/home/me/billing","name":"acme/billing"}`)

	if response.Code != http.StatusOK {
		t.Fatalf("got %d, want 200", response.Code)
	}
	if reader.registered.Path != "/home/me/billing" || reader.registered.Name != "acme/billing" {
		t.Errorf("registered %+v, want what was asked for", reader.registered)
	}
}

func TestCoveringNowhereIsRefused(t *testing.T) {
	server := New(&stubReader{}, stubSupervisor{}, "dev", "")

	response := body(t, server, http.MethodPost, "/api/repositories", `{}`)

	if response.Code != http.StatusBadRequest {
		t.Errorf("got %d, want 400", response.Code)
	}
}

func TestADirectoryCanBeDropped(t *testing.T) {
	reader := &stubReader{}
	server := New(reader, stubSupervisor{}, "dev", "")

	response := body(t, server, http.MethodDelete, "/api/repositories?path=/home/me/billing", "")

	if response.Code != http.StatusOK {
		t.Fatalf("got %d, want 200", response.Code)
	}
	if reader.forgot != "/home/me/billing" {
		t.Errorf("forgot %q, want the directory named", reader.forgot)
	}
}

func TestIndexingReportsWhatWasRead(t *testing.T) {
	reader := &stubReader{indexed: []core.Indexed{{Repository: "acme/billing", Files: 12}}}
	server := New(reader, stubSupervisor{}, "dev", "")

	response := body(t, server, http.MethodPost, "/api/index", `{"repository":"acme/billing"}`)

	var done []core.Indexed
	decode(t, response, &done)
	if len(done) != 1 || done[0].Files != 12 {
		t.Errorf("got %+v, want what the core read", done)
	}
	if reader.askedFor != "acme/billing" {
		t.Errorf("indexed %q, want acme/billing", reader.askedFor)
	}
}

func TestIndexingEverythingSaysSo(t *testing.T) {
	reader := &stubReader{}
	server := New(reader, stubSupervisor{}, "dev", "")

	body(t, server, http.MethodPost, "/api/index", `{"everything":true}`)

	if !reader.askedEverything {
		t.Error("asked for one repository, want all of them")
	}
}

func TestKnowledgeIsRecordedAgainstARepository(t *testing.T) {
	reader := &stubReader{}
	server := New(reader, stubSupervisor{}, "dev", "")

	response := body(t, server, http.MethodPut, "/api/knowledge",
		`{"repository":"acme/billing","id":"retry","kind":"decision","summary":"Retry three times."}`)

	if response.Code != http.StatusOK {
		t.Fatalf("got %d, want 200", response.Code)
	}
	if reader.askedFor != "acme/billing" || reader.recorded.ID != "retry" {
		t.Errorf("recorded %+v against %q", reader.recorded, reader.askedFor)
	}
}

func TestKnowledgeWithNothingToSayIsRefused(t *testing.T) {
	server := New(&stubReader{}, stubSupervisor{}, "dev", "")

	response := body(t, server, http.MethodPut, "/api/knowledge",
		`{"repository":"acme/billing","id":"retry","kind":"decision"}`)

	if response.Code != http.StatusBadRequest {
		t.Errorf("got %d, want 400", response.Code)
	}
}

func TestNothingRecordedIsAnEmptyListRatherThanNull(t *testing.T) {
	server := New(&stubReader{}, stubSupervisor{}, "dev", "")

	response := call(t, server, "/api/knowledge?repository=acme/billing")

	if !strings.Contains(response.Body.String(), `"items":[]`) {
		t.Errorf("got %q, want an empty list", response.Body.String())
	}
}

// A browser will not tell a page the absolute path of a folder somebody picked,
// so the agent lists the machine and the page navigates what it lists.
func TestBrowsingListsDirectoriesToPickFrom(t *testing.T) {
	server := New(&stubReader{}, stubSupervisor{}, "dev", "")

	response := call(t, server, "/api/browse?path="+t.TempDir())

	if response.Code != http.StatusOK {
		t.Fatalf("got %d, want 200", response.Code)
	}
	if !strings.Contains(response.Body.String(), `"entries"`) {
		t.Errorf("got %q, want a listing", response.Body.String())
	}
}

func TestBrowsingNowhereSaysSo(t *testing.T) {
	server := New(&stubReader{}, stubSupervisor{}, "dev", "")

	response := call(t, server, "/api/browse?path=/definitely/not/here")

	if response.Code != http.StatusNotFound {
		t.Errorf("got %d, want 404", response.Code)
	}
}

// The list of models is the core's, because the thing that would make the call
// is the thing that knows what it can call.
func TestTheModelsThisMachineCanNameAreListed(t *testing.T) {
	reader := &stubReader{offered: []core.Offering{
		{Provider: "anthropic", Models: []string{"anthropic/claude-sonnet-4-5"}},
	}}
	server := New(reader, stubSupervisor{}, "dev", "")

	response := call(t, server, "/api/models")

	if response.Code != http.StatusOK {
		t.Fatalf("got %d, want 200", response.Code)
	}
	if !strings.Contains(response.Body.String(), "anthropic/claude-sonnet-4-5") {
		t.Errorf("the model is missing from %s", response.Body.String())
	}
}

func TestAMachineWithNoModelsAnswersAnEmptyList(t *testing.T) {
	server := New(&stubReader{}, stubSupervisor{}, "dev", "")

	response := call(t, server, "/api/models")

	if strings.TrimSpace(response.Body.String()) != "[]" {
		t.Errorf("got %s, want an empty list a screen can draw", response.Body.String())
	}
}

func TestAKeyIsCheckedAgainstTheProviderRatherThanGuessedAt(t *testing.T) {
	reader := &stubReader{usable: core.Usable{Usable: false, Reason: "No access to that model."}}
	server := New(reader, stubSupervisor{}, "dev", "")

	response := body(t, server, http.MethodPost, "/api/models/check",
		`{"model":"anthropic/one","api_key":"sk-test"}`)

	if response.Code != http.StatusOK {
		t.Fatalf("got %d, want 200", response.Code)
	}
	if reader.askedModel != "anthropic/one" || reader.askedKey != "sk-test" {
		t.Errorf("asked about %q with %q, want what was sent", reader.askedModel, reader.askedKey)
	}
	if !strings.Contains(response.Body.String(), "No access to that model.") {
		t.Errorf("the reason is missing from %s", response.Body.String())
	}
}
