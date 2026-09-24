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
	"net"
	"net/http"
	"strconv"
	"strings"
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
	Nodes(ctx context.Context, repository string, opts core.NodeOptions) (core.NodePage, error)
	Attention(ctx context.Context, repository string) (core.Attention, error)
	Register(ctx context.Context, path, name string) (core.Repository, error)
	Forget(ctx context.Context, path string) error
	Index(ctx context.Context, repository string, everything, update bool) ([]core.Indexed, error)
	Knowledge(ctx context.Context, repository string, limit, offset int) (core.KnowledgePage, error)
	RecordKnowledge(ctx context.Context, repository string, item core.Knowledge) (core.Knowledge, error)
	ForgetKnowledge(ctx context.Context, repository, id string) error
	Initialize(ctx context.Context, repository string, dryRun, useModel bool) (core.Seeded, error)
	Skills(ctx context.Context, repository string) (core.SkillPage, error)
	Skill(ctx context.Context, id, repository string) (core.Skill, error)
	RecordSkill(ctx context.Context, stated core.Stated) (core.Skill, error)
	ForgetSkill(ctx context.Context, repository, scope, id string) error
	Review(ctx context.Context, ask core.Ask) (core.Reading, error)
	Reviewed(ctx context.Context, id string) (core.Reading, error)
	Reviews(ctx context.Context, repository string) ([]core.Reading, error)
	Settings(ctx context.Context) ([]core.Setting, error)
	Uses(ctx context.Context) ([]core.Use, error)
	Models(ctx context.Context) ([]core.Offering, error)
	CheckModel(ctx context.Context, model, key, baseURL string) (core.Usable, error)
	SetSetting(ctx context.Context, key string, value any) (core.Setting, error)
	ResetSetting(ctx context.Context, key string) (core.Setting, error)
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
	stop       func(context.Context) error
}

// New builds the agent's HTTP surface.
func New(reader Reader, supervisor Supervision, version, coreURL string) *Server {
	return &Server{
		reader:     reader,
		supervisor: supervisor,
		version:    version,
		coreURL:    coreURL,
	}
}

func (s *Server) SetStop(stop func(context.Context) error) { s.stop = stop }

func (s *Server) stopStack(w http.ResponseWriter, r *http.Request) {
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil || !net.ParseIP(host).IsLoopback() || r.Header.Get("Origin") != "" || r.Header.Get("X-Sourceant-Client") != "cli" {
		write(w, http.StatusForbidden, problem{Error: "stop is only available to the local CLI"})
		return
	}
	if s.stop == nil {
		write(w, http.StatusNotImplemented, problem{Error: "this agent does not support stopping"})
		return
	}
	if err := s.stop(r.Context()); err != nil {
		write(w, http.StatusInternalServerError, problem{Error: err.Error()})
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// Handler is the agent's routes.
func (s *Server) Handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("POST /api/stop", s.stopStack)
	mux.HandleFunc("GET /health", s.health)
	mux.HandleFunc("GET /api/repositories", s.repositories)
	mux.HandleFunc("POST /api/repositories", s.addRepository)
	mux.HandleFunc("DELETE /api/repositories", s.dropRepository)
	mux.HandleFunc("POST /api/index", s.index)
	mux.HandleFunc("GET /api/graph", s.graph)
	mux.HandleFunc("GET /api/architecture", s.architecture)
	mux.HandleFunc("POST /api/architecture/compare", s.compareArchitecture)
	mux.HandleFunc("GET /api/attention", s.attention)
	mux.HandleFunc("GET /api/knowledge", s.knowledge)
	mux.HandleFunc("PUT /api/knowledge", s.recordKnowledge)
	mux.HandleFunc("DELETE /api/knowledge", s.forgetKnowledge)
	mux.HandleFunc("POST /api/knowledge/initialize", s.initialize)
	mux.HandleFunc("GET /api/settings", s.settings)
	mux.HandleFunc("PUT /api/settings", s.setSetting)
	mux.HandleFunc("DELETE /api/settings", s.resetSetting)
	mux.HandleFunc("GET /api/nodes", s.nodes)
	mux.HandleFunc("GET /api/skills/uses", s.uses)
	mux.HandleFunc("GET /api/models", s.models)
	mux.HandleFunc("POST /api/models/check", s.checkModel)
	mux.HandleFunc("GET /api/skills", s.skills)
	mux.HandleFunc("GET /api/skills/{id...}", s.skill)
	mux.HandleFunc("PUT /api/skills", s.recordSkill)
	mux.HandleFunc("DELETE /api/skills", s.forgetSkill)
	mux.HandleFunc("POST /api/reviews", s.review)
	mux.HandleFunc("GET /api/reviews", s.listReviews)
	mux.HandleFunc("GET /api/reviews/{id}", s.reviewed)
	mux.HandleFunc("GET /api/browse", s.browse)
	mux.Handle("/mcp", s.mcp())
	mux.Handle("/mcp/", s.mcp())
	// Not method-scoped: Go refuses a "GET /" that is more general than a
	// method-agnostic "/mcp/" registered beside it.
	mux.Handle("/", ui.Handler())
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
	depth := 2
	if value := r.URL.Query().Get("depth"); value != "" {
		depth, err = strconv.Atoi(value)
		if err != nil || depth < 1 || depth > 5 {
			write(w, http.StatusBadRequest, problem{Error: "depth must be between 1 and 5"})
			return
		}
	}
	graph, err := s.reader.Graph(r.Context(), repository, core.GraphOptions{
		PathPrefix:   r.URL.Query().Get("path_prefix"),
		IncludeTests: r.URL.Query().Get("include_tests") == "true",
		NodeLimit:    limit,
		Focus:        r.URL.Query().Get("focus"),
		Depth:        depth,
		Query:        r.URL.Query().Get("q"),
	})
	if err != nil {
		fail(w, err)
		return
	}
	write(w, http.StatusOK, graph)
}

// A page of nodes, for a screen that wants a count rather than a drawing.
func (s *Server) nodes(w http.ResponseWriter, r *http.Request) {
	repository := r.URL.Query().Get("repository")
	if repository == "" {
		write(w, http.StatusBadRequest, problem{Error: "name a repository"})
		return
	}
	limit, err := strconv.Atoi(r.URL.Query().Get("limit"))
	if err != nil {
		limit = 0
	}
	page, err := s.reader.Nodes(r.Context(), repository, core.NodeOptions{
		Labels:   r.URL.Query()["labels"],
		FilePath: r.URL.Query().Get("file_path"),
		Limit:    limit,
	})
	if err != nil {
		fail(w, err)
		return
	}
	if page.Nodes == nil {
		page.Nodes = []core.Node{}
	}
	write(w, http.StatusOK, page)
}

func (s *Server) attention(w http.ResponseWriter, r *http.Request) {
	repository := r.URL.Query().Get("repository")
	if repository == "" {
		write(w, http.StatusBadRequest, problem{Error: "name a repository"})
		return
	}
	found, err := s.reader.Attention(r.Context(), repository)
	if err != nil {
		fail(w, err)
		return
	}
	// A repository with no recent history is an empty list, never a null, so a
	// screen can say "nothing yet" without special-casing the absent case.
	if found.Files == nil {
		found.Files = []core.Worth{}
	}
	write(w, http.StatusOK, found)
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

// knowledge is what is recorded, about one repository or about every one.
//
// Naming none is not a mistake here, unlike writing: a decision is remembered
// by what it decided rather than by which checkout it was filed against.
func (s *Server) knowledge(w http.ResponseWriter, r *http.Request) {
	repository := r.URL.Query().Get("repository")
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

func (s *Server) initialize(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Repository string `json:"repository"`
		DryRun     bool   `json:"dry_run"`
		UseModel   bool   `json:"use_model"`
	}
	if !readBody(w, r, &body) {
		return
	}
	if body.Repository == "" {
		write(w, http.StatusBadRequest, problem{Error: "name a repository"})
		return
	}
	seeded, err := s.reader.Initialize(r.Context(), body.Repository, body.DryRun, body.UseModel)
	if err != nil {
		fail(w, err)
		return
	}
	if seeded.Found == nil {
		seeded.Found = []core.Seed{}
	}
	write(w, http.StatusOK, seeded)
}

func (s *Server) skills(w http.ResponseWriter, r *http.Request) {
	page, err := s.reader.Skills(r.Context(), r.URL.Query().Get("repository"))
	if err != nil {
		fail(w, err)
		return
	}
	// A machine with no skills folder is an empty list, never a null, so a
	// screen can say "nothing yet" without special-casing the absent case.
	if page.Skills == nil {
		page.Skills = []core.Skill{}
	}
	write(w, http.StatusOK, page)
}

func (s *Server) skill(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if id == "" {
		write(w, http.StatusBadRequest, problem{Error: "name a skill"})
		return
	}
	found, err := s.reader.Skill(r.Context(), id, r.URL.Query().Get("repository"))
	if err != nil {
		fail(w, err)
		return
	}
	write(w, http.StatusOK, found)
}

func (s *Server) recordSkill(w http.ResponseWriter, r *http.Request) {
	var stated core.Stated
	if !readBody(w, r, &stated) {
		return
	}
	if stated.ID == "" {
		write(w, http.StatusBadRequest, problem{Error: "name a skill"})
		return
	}
	written, err := s.reader.RecordSkill(r.Context(), stated)
	if err != nil {
		fail(w, err)
		return
	}
	write(w, http.StatusOK, written)
}

func (s *Server) forgetSkill(w http.ResponseWriter, r *http.Request) {
	repository := r.URL.Query().Get("repository")
	scope := r.URL.Query().Get("scope")
	id := r.URL.Query().Get("id")
	if id == "" {
		write(w, http.StatusBadRequest, problem{Error: "name a skill"})
		return
	}
	if err := s.reader.ForgetSkill(r.Context(), repository, scope, id); err != nil {
		fail(w, err)
		return
	}
	write(w, http.StatusOK, map[string]string{"id": id})
}

func (s *Server) settings(w http.ResponseWriter, r *http.Request) {
	settings, err := s.reader.Settings(r.Context())
	if err != nil {
		fail(w, err)
		return
	}
	if settings == nil {
		settings = []core.Setting{}
	}
	write(w, http.StatusOK, settings)
}

func (s *Server) setSetting(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Key   string `json:"key"`
		Value any    `json:"value"`
	}
	if !readBody(w, r, &body) {
		return
	}
	if body.Key == "" {
		write(w, http.StatusBadRequest, problem{Error: "name a setting"})
		return
	}
	setting, err := s.reader.SetSetting(r.Context(), body.Key, body.Value)
	if err != nil {
		fail(w, err)
		return
	}
	write(w, http.StatusOK, setting)
}

func (s *Server) uses(w http.ResponseWriter, r *http.Request) {
	offered, err := s.reader.Uses(r.Context())
	if err != nil {
		fail(w, err)
		return
	}
	if offered == nil {
		offered = []core.Use{}
	}
	write(w, http.StatusOK, offered)
}

func (s *Server) models(w http.ResponseWriter, r *http.Request) {
	offered, err := s.reader.Models(r.Context())
	if err != nil {
		fail(w, err)
		return
	}
	// None is an empty list, never a null, so a screen can draw it.
	if offered == nil {
		offered = []core.Offering{}
	}
	write(w, http.StatusOK, offered)
}

func (s *Server) checkModel(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Model   string `json:"model"`
		APIKey  string `json:"api_key"`
		BaseURL string `json:"base_url"`
	}
	if !readBody(w, r, &body) {
		return
	}
	answer, err := s.reader.CheckModel(r.Context(), body.Model, body.APIKey, body.BaseURL)
	if err != nil {
		fail(w, err)
		return
	}
	write(w, http.StatusOK, answer)
}

func (s *Server) resetSetting(w http.ResponseWriter, r *http.Request) {
	key := r.URL.Query().Get("key")
	if key == "" {
		write(w, http.StatusBadRequest, problem{Error: "name a setting"})
		return
	}
	setting, err := s.reader.ResetSetting(r.Context(), key)
	if err != nil {
		fail(w, err)
		return
	}
	write(w, http.StatusOK, setting)
}

func (s *Server) browse(w http.ResponseWriter, r *http.Request) {
	// A name to search for answers with what matches it, anywhere under home,
	// rather than with one directory's contents.
	if term := r.URL.Query().Get("q"); strings.TrimSpace(term) != "" {
		found, err := browse.Find(term)
		if err != nil {
			write(w, http.StatusNotFound, problem{Error: err.Error()})
			return
		}
		write(w, http.StatusOK, browse.Listing{Path: browse.Home(), Entries: found})
		return
	}

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
	shutdownDone := make(chan struct{})
	go func() {
		defer close(shutdownDone)
		<-ctx.Done()
		shutdown, cancel := context.WithTimeout(context.WithoutCancel(ctx), 15*time.Second)
		defer cancel()
		if err := server.Shutdown(shutdown); err != nil {
			_ = server.Close()
		}
	}()
	if err := server.ListenAndServe(); !errors.Is(err, http.ErrServerClosed) {
		return err
	}
	<-shutdownDone
	return nil
}
