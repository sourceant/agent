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

func TestAnOrdinaryCallGivesUpOnTheOrdinaryDeadline(t *testing.T) {
	server := slow(300*time.Millisecond, []Repository{})
	defer server.Close()
	client := New(server.URL, 50*time.Millisecond)

	if _, err := client.Repositories(context.Background()); err == nil {
		t.Error("waited past the ordinary deadline")
	}
}

// Whoever asked owns the deadline. A caller who says fifty milliseconds gets
// fifty milliseconds, whatever the call would otherwise have waited.
func TestADeadlineAlreadySetIsNotReplaced(t *testing.T) {
	server := slow(300*time.Millisecond, []Repository{})
	defer server.Close()
	client := New(server.URL, Working)

	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	if _, err := client.Repositories(ctx); err == nil {
		t.Error("waited past the deadline it was given")
	}
}
