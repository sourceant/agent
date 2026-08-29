package api

import (
	"net/http"

	"github.com/sourceant/agent/internal/core"
)

// A review is kept by the core, not held here.
//
// The thing that asks for one is often not the thing that reads it: an agent
// runs one over MCP while somebody is in the middle of something else and
// hands them a link. A link into this process's memory stops working when this
// process restarts, which is not a property a link should have.

func (s *Server) review(w http.ResponseWriter, r *http.Request) {
	var ask core.Ask
	if !readBody(w, r, &ask) {
		return
	}
	if ask.Repository == "" {
		write(w, http.StatusBadRequest, problem{Error: "name a repository"})
		return
	}
	started, err := s.reader.Review(r.Context(), ask)
	if err != nil {
		fail(w, err)
		return
	}
	write(w, http.StatusAccepted, started)
}

func (s *Server) reviewed(w http.ResponseWriter, r *http.Request) {
	found, err := s.reader.Reviewed(r.Context(), r.PathValue("id"))
	if err != nil {
		fail(w, err)
		return
	}
	found.Review = drawable(found.Review)
	write(w, http.StatusOK, found)
}

func (s *Server) listReviews(w http.ResponseWriter, r *http.Request) {
	found, err := s.reader.Reviews(r.Context(), r.URL.Query().Get("repository"))
	if err != nil {
		fail(w, err)
		return
	}
	// None yet is an empty list, never a null, so a screen can draw it.
	if found == nil {
		found = []core.Reading{}
	}
	write(w, http.StatusOK, found)
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
	if reviewed.Read.Suggestions == nil {
		reviewed.Read.Suggestions = []core.Suggestion{}
	}
	return reviewed
}
