package api

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"net/http"
	"sort"
	"sync"
	"time"

	"github.com/sourceant/agent/internal/core"
)

// A review asks a model about several rules and takes tens of seconds, and it
// will take longer on a bigger repository. Answering it on the request that
// started it means holding a browser connection open for all of that, and
// anything that interrupts the connection loses work that had already been
// paid for.
//
// So the agent runs it and remembers the answer. The page starts one and asks
// how it went, which survives a slow provider, a reload, and a person walking
// away from the screen.

const (
	Running = "running"
	Done    = "done"
	Failed  = "failed"
)

// Kept so a page that reloads can still collect an answer, and bounded so a
// long-lived agent does not accumulate every review anybody ever ran.
const remembered = 20

type job struct {
	ID      string      `json:"id"`
	Status  string      `json:"status"`
	Error   string      `json:"error,omitempty"`
	Review  core.Review `json:"review"`
	Started time.Time   `json:"started"`
}

type reviews struct {
	mutex sync.Mutex
	jobs  map[string]*job
}

func newReviews() *reviews {
	return &reviews{jobs: map[string]*job{}}
}

func name() string {
	raw := make([]byte, 8)
	if _, err := rand.Read(raw); err != nil {
		return hex.EncodeToString([]byte(time.Now().Format(time.RFC3339Nano)))
	}
	return hex.EncodeToString(raw)
}

func (r *reviews) start() *job {
	started := &job{ID: name(), Status: Running, Started: time.Now()}
	r.mutex.Lock()
	defer r.mutex.Unlock()
	r.jobs[started.ID] = started
	r.forget()
	return started
}

// forget drops the oldest once there are more than are worth keeping. The
// caller holds the lock.
func (r *reviews) forget() {
	if len(r.jobs) <= remembered {
		return
	}
	order := make([]*job, 0, len(r.jobs))
	for _, one := range r.jobs {
		order = append(order, one)
	}
	sort.Slice(order, func(a, b int) bool { return order[a].Started.Before(order[b].Started) })
	for _, one := range order[:len(order)-remembered] {
		delete(r.jobs, one.ID)
	}
}

func (r *reviews) finish(id string, reviewed core.Review, err error) {
	r.mutex.Lock()
	defer r.mutex.Unlock()
	one, ok := r.jobs[id]
	if !ok {
		return
	}
	if err != nil {
		one.Status, one.Error = Failed, err.Error()
		return
	}
	one.Status, one.Review = Done, drawable(reviewed)
}

func (r *reviews) get(id string) (job, bool) {
	r.mutex.Lock()
	defer r.mutex.Unlock()
	one, ok := r.jobs[id]
	if !ok {
		return job{}, false
	}
	return *one, true
}

// drawable fills in the empty lists, so a screen can draw an answer without
// special-casing every absent one.
func drawable(reviewed core.Review) core.Review {
	if reviewed.Changed == nil {
		reviewed.Changed = []core.ChangedFile{}
	}
	if reviewed.Skills == nil {
		reviewed.Skills = []core.Skill{}
	}
	if reviewed.Knowledge == nil {
		reviewed.Knowledge = []core.Recorded{}
	}
	if reviewed.Verdicts == nil {
		reviewed.Verdicts = []core.Verdict{}
	}
	return reviewed
}

func (s *Server) review(w http.ResponseWriter, r *http.Request) {
	var ask core.Ask
	if !readBody(w, r, &ask) {
		return
	}
	if ask.Repository == "" {
		write(w, http.StatusBadRequest, problem{Error: "name a repository"})
		return
	}

	started := s.reviews.start()
	go func() {
		// Not the request's context: that one ends when this handler returns,
		// which is immediately.
		ctx, done := context.WithTimeout(context.Background(), core.Working)
		defer done()
		reviewed, err := s.reader.Review(ctx, ask)
		s.reviews.finish(started.ID, reviewed, err)
	}()

	write(w, http.StatusAccepted, job{ID: started.ID, Status: Running, Started: started.Started})
}

func (s *Server) reviewed(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	one, ok := s.reviews.get(id)
	if !ok {
		write(w, http.StatusNotFound, problem{
			Error: "That review is not one this agent is holding. Start it again.",
		})
		return
	}
	if one.Status == Done {
		one.Review = drawable(one.Review)
	}
	write(w, http.StatusOK, one)
}
