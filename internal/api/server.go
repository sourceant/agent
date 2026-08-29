// Package api is the agent's own HTTP surface.
//
// Everything else talks to the agent rather than to the Python core: the agent
// is the process that is always up, it knows which port the core landed on, and
// it is where a view across every registered repository will be assembled. A
// client that reached past it would have to learn all three.
package api

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"time"

	"github.com/sourceant/agent/internal/browse"
	"github.com/sourceant/agent/internal/core"
	"github.com/sourceant/agent/internal/ui"
)

// Reader is the part of the core client this server needs.
type Reader interface {
	Healthy(ctx context.Context) bool
	Repositories(ctx context.Context) ([]core.Repository, error)
	Graph(ctx context.Context, repository string, opts core.GraphOptions) (core.Graph, error)
	Register(ctx context.Context, path, name string) (core.Repository, error)
	Forget(ctx context.Context, path string) error
	Index(ctx context.Context, repository string, everything, update bool) ([]core.Indexed, error)
	Knowledge(ctx context.Context, repository string, limit, offset int) (core.KnowledgePage, error)
	RecordKnowledge(ctx context.Context, repository string, item core.Knowledge) (core.Knowledge, error)
	ForgetKnowledge(ctx context.Context, repository, id string) error
}

// Supervision is the part of the supervisor this server reports on.
type Supervision interface {
	Starts() int
	LastExit() error
}

// Status is what the agent says about itself.
type Status struct {
	Version string `json:"version"`
	// CoreURL is where the agent's core is listening.
	CoreURL string `json:"core_url"`
	// CoreUp is whether it answered just now.
	CoreUp bool `json:"core_up"`
	// CoreStarts counts launches, so a number that keeps climbing is a
	// core that keeps dying.
	CoreStarts int    `json:"core_starts"`
	LastExit   string `json:"last_exit,omitempty"`
}

// Server answers for the agent.
type Server struct {
	reader     Reader
	supervisor Supervision
	version    string
	coreURL    string
}

// New builds the agent's HTTP surface.
func New(reader Reader, supervisor Supervision, version, coreURL string) *Server {
	return &Server{reader: reader, supervisor: supervisor, version: version, coreURL: coreURL}
}

// Handler is the agent's routes.
func (s *Server) Handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /health", s.health)
	mux.HandleFunc("GET /api/repositories", s.repositories)
	mux.HandleFunc("POST /api/repositories", s.addRepository)
	mux.HandleFunc("DELETE /api/repositories", s.dropRepository)
	mux.HandleFunc("POST /api/index", s.index)
	mux.HandleFunc("GET /api/graph", s.graph)
	mux.HandleFunc("GET /api/knowledge", s.knowledge)
	mux.HandleFunc("PUT /api/knowledge", s.recordKnowledge)
	mux.HandleFunc("DELETE /api/knowledge", s.forgetKnowledge)
	mux.HandleFunc("GET /api/browse", s.browse)
	mux.Handle("GET /", ui.Handler())
	return mux
}

func (s *Server) health(w http.ResponseWriter, r *http.Request) {
	status := Status{Version: s.version, CoreURL: s.coreURL}
	if s.reader != nil {
		status.CoreUp = s.reader.Healthy(r.Context())
	}
	if s.supervisor != nil {
		status.CoreStarts = s.supervisor.Starts()
		if exit := s.supervisor.LastExit(); exit != nil {
			status.LastExit = exit.Error()
		}
	}
	write(w, http.StatusOK, status)
}

func (s *Server) repositories(w http.ResponseWriter, r *http.Request) {
	repositories, err := s.reader.Repositories(r.Context())
	if err != nil {
		fail(w, err)
		return
	}
	// An empty registry is an empty list, never a null, so a client can draw
	// "nothing indexed yet" without special-casing the absent case.
	if repositories == nil {
		repositories = []core.Repository{}
	}
	write(w, http.StatusOK, repositories)
}

func (s *Server) graph(w http.ResponseWriter, r *http.Request) {
	repository := r.URL.Query().Get("repository")
	if repository == "" {
		write(w, http.StatusBadRequest, problem{Error: "name a repository"})
		return
	}
	limit, err := strconv.Atoi(r.URL.Query().Get("node_limit"))
	if err != nil {
		limit = 0
	}
	graph, err := s.reader.Graph(r.Context(), repository, core.GraphOptions{
		PathPrefix:   r.URL.Query().Get("path_prefix"),
		IncludeTests: r.URL.Query().Get("include_tests") == "true",
		NodeLimit:    limit,
	})
	if err != nil {
		fail(w, err)
		return
	}
	write(w, http.StatusOK, graph)
}

func (s *Server) addRepository(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Path string `json:"path"`
		Name string `json:"name"`
	}
	if !readBody(w, r, &body) {
		return
	}
	if body.Path == "" {
		write(w, http.StatusBadRequest, problem{Error: "name a directory"})
		return
	}
	added, err := s.reader.Register(r.Context(), body.Path, body.Name)
	if err != nil {
		fail(w, err)
		return
	}
	write(w, http.StatusOK, added)
}

func (s *Server) dropRepository(w http.ResponseWriter, r *http.Request) {
	path := r.URL.Query().Get("path")
	if path == "" {
		write(w, http.StatusBadRequest, problem{Error: "name a directory"})
		return
	}
	if err := s.reader.Forget(r.Context(), path); err != nil {
		fail(w, err)
		return
	}
	write(w, http.StatusOK, map[string]string{"path": path})
}

func (s *Server) index(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Repository string `json:"repository"`
		Everything bool   `json:"everything"`
		// Update reads only what changed. Asking for none of it means read it
		// again, which is what somebody pressing a button for it means.
		Update bool `json:"update"`
	}
	if !readBody(w, r, &body) {
		return
	}
	done, err := s.reader.Index(r.Context(), body.Repository, body.Everything, body.Update)
	if err != nil {
		fail(w, err)
		return
	}
	if done == nil {
		done = []core.Indexed{}
	}
	write(w, http.StatusOK, done)
}

func (s *Server) knowledge(w http.ResponseWriter, r *http.Request) {
	repository := r.URL.Query().Get("repository")
	if repository == "" {
		write(w, http.StatusBadRequest, problem{Error: "name a repository"})
		return
	}
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	offset, _ := strconv.Atoi(r.URL.Query().Get("offset"))
	page, err := s.reader.Knowledge(r.Context(), repository, limit, offset)
	if err != nil {
		fail(w, err)
		return
	}
	if page.Items == nil {
		page.Items = []core.Knowledge{}
	}
	write(w, http.StatusOK, page)
}

func (s *Server) recordKnowledge(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Repository string `json:"repository"`
		core.Knowledge
	}
	if !readBody(w, r, &body) {
		return
	}
	if body.Repository == "" || body.ID == "" || body.Summary == "" {
		write(w, http.StatusBadRequest, problem{Error: "a repository, an id and a summary are needed"})
		return
	}
	recorded, err := s.reader.RecordKnowledge(r.Context(), body.Repository, body.Knowledge)
	if err != nil {
		fail(w, err)
		return
	}
	write(w, http.StatusOK, recorded)
}

func (s *Server) forgetKnowledge(w http.ResponseWriter, r *http.Request) {
	repository := r.URL.Query().Get("repository")
	id := r.URL.Query().Get("id")
	if repository == "" || id == "" {
		write(w, http.StatusBadRequest, problem{Error: "a repository and an id are needed"})
		return
	}
	if err := s.reader.ForgetKnowledge(r.Context(), repository, id); err != nil {
		fail(w, err)
		return
	}
	write(w, http.StatusOK, map[string]string{"id": id})
}

func (s *Server) browse(w http.ResponseWriter, r *http.Request) {
	listing, err := browse.At(r.URL.Query().Get("path"))
	if err != nil {
		write(w, http.StatusNotFound, problem{Error: err.Error()})
		return
	}
	write(w, http.StatusOK, listing)
}

func readBody(w http.ResponseWriter, r *http.Request, into any) bool {
	if err := json.NewDecoder(r.Body).Decode(into); err != nil {
		write(w, http.StatusBadRequest, problem{Error: "the body is not readable as JSON"})
		return false
	}
	return true
}

type problem struct {
	Error string `json:"error"`
}

// fail answers with the core's own status where it gave one, so a repository
// nobody registered reads as 404 here too rather than as the agent breaking.
func fail(w http.ResponseWriter, err error) {
	var coreError *core.Error
	if errors.As(err, &coreError) {
		write(w, coreError.StatusCode, problem{Error: coreError.Detail})
		return
	}
	write(w, http.StatusBadGateway, problem{Error: err.Error()})
}

func write(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(body)
}

// Serve runs the agent's HTTP surface until ctx is cancelled.
func (s *Server) Serve(ctx context.Context, address string) error {
	server := &http.Server{
		Addr:              address,
		Handler:           s.Handler(),
		ReadHeaderTimeout: 10 * time.Second,
	}
	go func() {
		<-ctx.Done()
		shutdown, cancel := context.WithTimeout(context.WithoutCancel(ctx), 5*time.Second)
		defer cancel()
		_ = server.Shutdown(shutdown)
	}()
	if err := server.ListenAndServe(); !errors.Is(err, http.ErrServerClosed) {
		return err
	}
	return nil
}
