package api

import (
	"context"
	"encoding/json"
	"net/http"
	"strconv"
)

type architectureReader interface {
	Architecture(context.Context, string, int, bool) (json.RawMessage, error)
	CompareArchitecture(context.Context, json.RawMessage) (json.RawMessage, error)
}

func (s *Server) architecture(w http.ResponseWriter, r *http.Request) {
	reader, ok := s.reader.(architectureReader)
	if !ok {
		write(w, http.StatusNotImplemented, problem{Error: "This core does not support architecture readings"})
		return
	}
	repository := r.URL.Query().Get("repository")
	depth := 1
	var err error
	if given := r.URL.Query().Get("depth"); given != "" {
		depth, err = strconv.Atoi(given)
	}
	if repository == "" || err != nil || depth < 1 || depth > 4 {
		write(w, http.StatusBadRequest, problem{Error: "Name a repository and a depth between 1 and 4"})
		return
	}
	includeTests := false
	if given := r.URL.Query().Get("include_tests"); given != "" {
		includeTests, err = strconv.ParseBool(given)
		if err != nil {
			write(w, http.StatusBadRequest, problem{Error: "include_tests must be true or false"})
			return
		}
	}
	result, err := reader.Architecture(r.Context(), repository, depth, includeTests)
	if err != nil {
		fail(w, err)
		return
	}
	write(w, http.StatusOK, result)
}

func (s *Server) compareArchitecture(w http.ResponseWriter, r *http.Request) {
	reader, ok := s.reader.(architectureReader)
	if !ok {
		write(w, http.StatusNotImplemented, problem{Error: "This core does not support architecture comparisons"})
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, 8<<20)
	var baseline json.RawMessage
	if !readBody(w, r, &baseline) {
		return
	}
	result, err := reader.CompareArchitecture(r.Context(), baseline)
	if err != nil {
		fail(w, err)
		return
	}
	write(w, http.StatusOK, result)
}
