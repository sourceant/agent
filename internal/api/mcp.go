package api

import (
	"net/http"
	"net/http/httputil"
	"net/url"
	"strings"
)

// mcp proxies the MCP endpoint through to the core.
//
// The agent is the address a client is given: it is always up, and it knows
// which port the core landed on this time. A client pointed straight at the
// core would have to be reconfigured every restart.
//
// FlushInterval is -1 because a streamable HTTP response is written as it is
// produced. Buffered, the client waits for a response the server considers
// already sent, and the call hangs.
func (s *Server) mcp() http.Handler {
	target, err := url.Parse(s.coreURL)
	if err != nil {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			http.Error(w, "the core address is not a URL", http.StatusBadGateway)
		})
	}

	proxy := httputil.NewSingleHostReverseProxy(target)
	proxy.FlushInterval = -1
	proxy.ErrorHandler = func(w http.ResponseWriter, r *http.Request, err error) {
		http.Error(w, "the core is not answering", http.StatusBadGateway)
	}

	forward := proxy.Director
	proxy.Director = func(r *http.Request) {
		forward(r)
		// Rewritten so the core sees its own address rather than the agent's.
		// The core refuses a Host it does not recognise, which is what keeps an
		// unauthenticated endpoint off anything but loopback.
		r.Host = target.Host
		if !strings.HasPrefix(r.URL.Path, "/mcp") {
			r.URL.Path = "/mcp" + r.URL.Path
		}
	}
	return proxy
}
