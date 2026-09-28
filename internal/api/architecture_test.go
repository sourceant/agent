package api

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/sourceant/agent/internal/core"
)

func TestArchitectureThroughAgent(t *testing.T) {
	snapshot, err := os.ReadFile("../core/testdata/architecture.json")
	if err != nil {
		t.Fatal(err)
	}
	comparison, err := os.ReadFile("../core/testdata/architecture-comparison.json")
	if err != nil {
		t.Fatal(err)
	}
	calls := 0
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		result := snapshot
		switch r.URL.Path {
		case "/api/code/architecture":
			if r.URL.Query().Get("repository") != "acme/billing" || r.URL.Query().Get("depth") != "2" {
				t.Errorf("wrong query: %s", r.URL.RawQuery)
			}
		case "/api/code/architecture/compare":
			if r.Method != http.MethodPost {
				t.Errorf("wrong method: %s", r.Method)
			}
			body, _ := io.ReadAll(r.Body)
			var actual, expected any
			_ = json.Unmarshal(body, &actual)
			_ = json.Unmarshal(snapshot, &expected)
			a, _ := json.Marshal(actual)
			b, _ := json.Marshal(expected)
			if !bytes.Equal(a, b) {
				t.Error("the baseline changed in transit")
			}
			result = comparison
		default:
			t.Errorf("unexpected upstream route: %s", r.URL.Path)
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"status": "success", "data": json.RawMessage(result)})
	}))
	defer upstream.Close()
	handler := New(core.New(upstream.URL, time.Second), nil, "test", upstream.URL).Handler()
	for _, query := range []string{"", "?repository=acme/billing&depth=0", "?repository=acme/billing&include_tests=wrong"} {
		response := httptest.NewRecorder()
		handler.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/api/architecture"+query, nil))
		if response.Code != http.StatusBadRequest {
			t.Fatalf("invalid request returned %d", response.Code)
		}
	}
	if calls != 0 {
		t.Fatal("invalid requests reached the core")
	}
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/api/architecture?repository=acme/billing&depth=2", nil))
	if response.Code != http.StatusOK || !bytes.Contains(response.Body.Bytes(), []byte("payments")) {
		t.Fatalf("reading: %d %s", response.Code, response.Body.String())
	}
	response = httptest.NewRecorder()
	handler.ServeHTTP(response, httptest.NewRequest(http.MethodPost, "/api/architecture/compare", bytes.NewReader(snapshot)))
	if response.Code != http.StatusOK {
		t.Fatalf("comparison: %d %s", response.Code, response.Body.String())
	}
	if calls != 2 {
		t.Fatalf("got %d upstream requests, want 2", calls)
	}
}

func TestArchitecturePreservesCoreDenial(t *testing.T) {
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusForbidden)
		_, _ = w.Write([]byte(`{"detail":"local access only"}`))
	}))
	defer upstream.Close()
	handler := New(core.New(upstream.URL, time.Second), nil, "test", upstream.URL).Handler()
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/api/architecture?repository=acme/billing", nil))
	if response.Code != http.StatusForbidden {
		t.Fatalf("got %d", response.Code)
	}
}
