package api

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestStopRequiresLocalCLI(t *testing.T) {
	for _, tc := range []struct {
		name, remote, origin, client string
		want                         int
	}{
		{"local", "127.0.0.1:1234", "", "cli", 204},
		{"ipv6", "[::1]:1234", "", "cli", 204},
		{"remote", "192.0.2.1:1234", "", "cli", 403},
		{"browser", "127.0.0.1:1234", "http://example.org", "cli", 403},
		{"missing client", "127.0.0.1:1234", "", "", 403},
	} {
		t.Run(tc.name, func(t *testing.T) {
			server := New(nil, nil, "test", "")
			stopped := false
			server.SetStop(func(context.Context) error { stopped = true; return nil })
			req := httptest.NewRequest(http.MethodPost, "/api/stop", nil)
			req.RemoteAddr = tc.remote
			req.Header.Set("Origin", tc.origin)
			req.Header.Set("X-Sourceant-Client", tc.client)
			response := httptest.NewRecorder()
			server.Handler().ServeHTTP(response, req)
			if response.Code != tc.want || stopped != (tc.want == 204) {
				t.Fatalf("status=%d stopped=%v", response.Code, stopped)
			}
		})
	}
}
