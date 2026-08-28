package core

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"
)

// The fixtures are answers captured from a running core, not written here, so a
// change to what it serves fails these rather than passing against our guess.
func fixture(t *testing.T, name string) []byte {
	t.Helper()
	data, err := os.ReadFile(filepath.Join("testdata", name))
	if err != nil {
		t.Fatalf("reading fixture: %v", err)
	}
	return data
}

func serving(t *testing.T, routes map[string]func(http.ResponseWriter, *http.Request)) *Client {
	t.Helper()
	mux := http.NewServeMux()
	for pattern, handler := range routes {
		mux.HandleFunc(pattern, handler)
	}
	server := httptest.NewServer(mux)
	t.Cleanup(server.Close)
	return New(server.URL, 5*time.Second)
}

func TestRepositoriesReadsWhatTheCoreRegistered(t *testing.T) {
	client := serving(t, map[string]func(http.ResponseWriter, *http.Request){
		"/api/code/repositories": func(w http.ResponseWriter, _ *http.Request) {
			_, _ = w.Write(fixture(t, "repositories.json"))
		},
	})

	repositories, err := client.Repositories(context.Background())
	if err != nil {
		t.Fatalf("listing repositories: %v", err)
	}

	if len(repositories) != 1 {
		t.Fatalf("got %d repositories, want 1", len(repositories))
	}
	if repositories[0].Name != "local/sourceant" {
		t.Errorf("got name %q, want local/sourceant", repositories[0].Name)
	}
	if repositories[0].Path == "" {
		t.Error("got an empty path, want the directory the core registered")
	}
}

func TestGraphKeepsWhatTellsAFileFromAFunction(t *testing.T) {
	client := serving(t, map[string]func(http.ResponseWriter, *http.Request){
		"/api/code/graph": func(w http.ResponseWriter, _ *http.Request) {
			_, _ = w.Write(fixture(t, "graph.json"))
		},
	})

	graph, err := client.Graph(context.Background(), "local/sourceant", GraphOptions{})
	if err != nil {
		t.Fatalf("reading graph: %v", err)
	}

	if len(graph.Nodes) == 0 || len(graph.Links) == 0 {
		t.Fatalf("got %d nodes and %d links, want both", len(graph.Nodes), len(graph.Links))
	}
	if graph.Truncated {
		t.Error("got a truncated graph, want the whole captured scope")
	}

	var file, symbol *Node
	for i := range graph.Nodes {
		switch {
		case file == nil && slices.Contains(graph.Nodes[i].Labels, "File"):
			file = &graph.Nodes[i]
		case symbol == nil && graph.Nodes[i].Kind == "function":
			symbol = &graph.Nodes[i]
		}
	}
	if file == nil || symbol == nil {
		t.Fatal("the captured graph holds no file and function to tell apart")
	}
	// A file is a file whatever it is written in. Colouring Python files apart
	// from Go ones would be colouring the wrong question, so the language sits
	// beside the kind rather than standing in for it.
	if file.Kind != "file" || file.Language != "python" {
		t.Errorf("got kind %q language %q, want file and python", file.Kind, file.Language)
	}
	if file.Path == "" || symbol.Path == "" {
		t.Error("got a node with no path, want where the code sits")
	}
}

// A drawing sizes a node by how busy it is and colours it by which part of the
// repository it belongs to, so neither may be missing from what is read.
func TestGraphKeepsWhatSizesAndColoursANode(t *testing.T) {
	client := serving(t, map[string]func(http.ResponseWriter, *http.Request){
		"/api/code/graph": func(w http.ResponseWriter, _ *http.Request) {
			_, _ = w.Write(fixture(t, "graph.json"))
		},
	})

	graph, err := client.Graph(context.Background(), "local/sourceant", GraphOptions{})
	if err != nil {
		t.Fatalf("reading graph: %v", err)
	}

	if len(graph.Communities) == 0 {
		t.Fatal("the captured graph found no parts to colour")
	}
	for _, part := range graph.Communities {
		if part.Name == "" || part.Size == 0 {
			t.Errorf("got part %+v, want one saying what it is and how big", part)
		}
	}

	busiest, coloured := 0, 0
	for _, node := range graph.Nodes {
		if node.Degree > busiest {
			busiest = node.Degree
		}
		if node.Community != nil {
			coloured++
		}
	}
	if busiest == 0 {
		t.Error("no node says how many lines meet at it")
	}
	if coloured == 0 {
		t.Error("no node says which part it belongs to")
	}
}

func TestGraphPassesOnWhatNarrowsADrawing(t *testing.T) {
	var asked string
	client := serving(t, map[string]func(http.ResponseWriter, *http.Request){
		"/api/code/graph": func(w http.ResponseWriter, r *http.Request) {
			asked = r.URL.RawQuery
			_, _ = w.Write(fixture(t, "graph.json"))
		},
	})

	_, err := client.Graph(context.Background(), "local/sourceant", GraphOptions{
		PathPrefix:   "src/config/",
		IncludeTests: true,
		NodeLimit:    120,
	})
	if err != nil {
		t.Fatalf("reading graph: %v", err)
	}

	for _, want := range []string{
		"repository=local%2Fsourceant",
		"path_prefix=src%2Fconfig%2F",
		"include_tests=true",
		"node_limit=120",
	} {
		if !strings.Contains(asked, want) {
			t.Errorf("query %q is missing %q", asked, want)
		}
	}
}

func TestAnUnregisteredRepositoryIsReportedAsSuch(t *testing.T) {
	client := serving(t, map[string]func(http.ResponseWriter, *http.Request){
		"/api/code/graph": func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusNotFound)
			_, _ = w.Write(fixture(t, "unknown_repository.json"))
		},
	})

	_, err := client.Graph(context.Background(), "acme/nope", GraphOptions{})

	var coreError *Error
	if !errors.As(err, &coreError) {
		t.Fatalf("got %v, want a core error", err)
	}
	if !coreError.NotFound() {
		t.Errorf("got status %d, want 404", coreError.StatusCode)
	}
	if !strings.Contains(coreError.Error(), "not registered on this machine") {
		t.Errorf("got %q, want the reason the core gave", coreError.Error())
	}
}

func TestHealthyIsFalseWhenTheCoreIsNotThere(t *testing.T) {
	client := New("http://127.0.0.1:1", 200*time.Millisecond)

	if client.Healthy(context.Background()) {
		t.Error("reported a core that is not listening as healthy")
	}
}
