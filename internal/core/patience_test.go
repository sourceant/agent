package core

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func slow(delay time.Duration, body any) *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		select {
		case <-time.After(delay):
		case <-r.Context().Done():
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{"data": body})
	}))
}

// A review asks a model about several rules one at a time, which is minutes.
// The ordinary deadline is seconds, and applying it on top of the longer one
// cut every review off before the first answer came back.
func TestAskingAModelOutlastsTheOrdinaryDeadline(t *testing.T) {
	server := slow(300*time.Millisecond, map[string]any{"ready": true})
	defer server.Close()
	client := New(server.URL, 50*time.Millisecond)

	reviewed, err := client.Review(context.Background(), Ask{
		Repository: "acme/billing",
		UseModel:   true,
	})

	if err != nil {
		t.Fatalf("reviewing: %v", err)
	}
	if !reviewed.Ready {
		t.Error("the answer did not come back")
	}
}

func TestAnOrdinaryCallGivesUpOnTheOrdinaryDeadline(t *testing.T) {
	server := slow(300*time.Millisecond, []Repository{})
	defer server.Close()
	client := New(server.URL, 50*time.Millisecond)

	if _, err := client.Repositories(context.Background()); err == nil {
		t.Error("waited past the ordinary deadline")
	}
}

// Whoever asked owns the deadline. A caller who says thirty seconds gets thirty
// seconds even from a call that would otherwise wait fifteen minutes.
func TestADeadlineAlreadySetIsNotReplaced(t *testing.T) {
	server := slow(300*time.Millisecond, map[string]any{"ready": true})
	defer server.Close()
	client := New(server.URL, Patience)

	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	if _, err := client.Review(ctx, Ask{Repository: "acme/billing", UseModel: true}); err == nil {
		t.Error("waited past the deadline it was given")
	}
}
