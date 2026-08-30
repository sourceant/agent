package ui

import (
	"net/http"
	"net/http/httptest"
	"regexp"
	"strings"
	"testing"
)

func fetch(t *testing.T, path string) *httptest.ResponseRecorder {
	t.Helper()
	recorder := httptest.NewRecorder()
	Handler().ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, path, nil))
	return recorder
}

// The page is built by Vite, so the names are hashed and change every build.
// What has to hold is that whatever the page asks for is embedded beside it:
// the agent serves a machine that may have no network, and a missing chunk
// there is a blank page nobody finds out about until it is opened.
func TestEverythingThePageAsksForIsEmbeddedBesideIt(t *testing.T) {
	page := fetch(t, "/")
	if page.Code != http.StatusOK {
		t.Fatalf("got %d for the page, want 200", page.Code)
	}

	referenced := regexp.MustCompile(`(?:src|href)="\.?(/?assets/[^"]+|/?favicon\.svg)"`).
		FindAllStringSubmatch(page.Body.String(), -1)
	if len(referenced) < 2 {
		t.Fatalf("the page asks for almost nothing, which means it did not build:\n%s", page.Body.String())
	}

	for _, match := range referenced {
		asset := "/" + strings.TrimPrefix(match[1], "/")
		if response := fetch(t, asset); response.Code != http.StatusOK {
			t.Errorf("got %d for %s, want 200", response.Code, asset)
		}
	}
}

func TestTheBuildCarriesItsOwnGraphLibraries(t *testing.T) {
	page := fetch(t, "/")
	entry := regexp.MustCompile(`src="\.?(/?assets/index-[^"]+\.js)"`).
		FindStringSubmatch(page.Body.String())
	if entry == nil {
		t.Fatal("the page has no entry script")
	}

	body := fetch(t, "/"+strings.TrimPrefix(entry[1], "/")).Body.String()

	// Both renderers are pulled in dynamically, so the entry names their chunks
	// rather than containing them. Either way nothing is fetched from a CDN.
	for _, want := range []string{"force-graph", "3d-force-graph"} {
		if !strings.Contains(body, want) {
			t.Errorf("the entry script never reaches %s", want)
		}
	}
	if strings.Contains(body, "https://cdn") || strings.Contains(body, "unpkg.com") {
		t.Error("the page fetches something from a CDN, which a machine with no network cannot")
	}
}

func TestABrowserIsNotToldToKeepAViewTheNextAgentReplaces(t *testing.T) {
	if got := fetch(t, "/").Header().Get("Cache-Control"); got != "no-store" {
		t.Errorf("got Cache-Control %q, want no-store", got)
	}
}

// A link somebody was handed by an agent is a real path, not a fragment. The
// browser asks this server for it directly, so anything that is not a file has
// to answer with the page rather than 404.
func TestAPathThePageOwnsAnswersWithThePage(t *testing.T) {
	for _, path := range []string{"/reviews/abc123", "/settings", "/repositories"} {
		answered := fetch(t, path)
		if answered.Code != http.StatusOK {
			t.Fatalf("got %d for %s, want 200", answered.Code, path)
		}
		if !strings.Contains(answered.Body.String(), "id=\"app\"") {
			t.Fatalf("%s did not answer with the page", path)
		}
	}
}

// A missing asset still 404s. Answering with the page would hand a broken
// script tag an HTML document and fail somewhere less obvious.
func TestAMissingAssetIsStillMissing(t *testing.T) {
	answered := fetch(t, "/assets/nothing-here.js")
	if answered.Code != http.StatusNotFound {
		t.Fatalf("got %d for a missing asset, want 404", answered.Code)
	}
}
