package ui

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func fetch(t *testing.T, path string) *httptest.ResponseRecorder {
	t.Helper()
	recorder := httptest.NewRecorder()
	Handler().ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, path, nil))
	return recorder
}

// Everything the page pulls in has to be embedded, or the view is blank on a
// machine with no network and nobody finds out until it is opened.
func TestItServesEveryAssetThePageAsksFor(t *testing.T) {
	page := fetch(t, "/")
	if page.Code != http.StatusOK {
		t.Fatalf("got %d for the page, want 200", page.Code)
	}
	body := page.Body.String()

	for _, asset := range []string{"styles.css", "app.js", "vendor/force-graph.min.js", "favicon.svg"} {
		if !strings.Contains(body, asset) {
			t.Errorf("the page does not ask for %s", asset)
			continue
		}
		if response := fetch(t, "/"+asset); response.Code != http.StatusOK {
			t.Errorf("got %d for %s, want 200", response.Code, asset)
		}
	}
}

func TestTheGraphLibraryIsWholeRatherThanAStub(t *testing.T) {
	response := fetch(t, "/vendor/force-graph.min.js")

	if response.Body.Len() < 100_000 {
		t.Errorf("got %d bytes, want the whole library", response.Body.Len())
	}
	if !strings.Contains(response.Body.String(), "ForceGraph") {
		t.Error("the vendored file does not define ForceGraph")
	}
}

func TestABrowserIsNotToldToKeepAViewTheNextAgentReplaces(t *testing.T) {
	if got := fetch(t, "/").Header().Get("Cache-Control"); got != "no-store" {
		t.Errorf("got Cache-Control %q, want no-store", got)
	}
}
