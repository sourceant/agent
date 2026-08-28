// Package core reads the Python indexer's HTTP surface.
//
// The agent never parses code. Everything it knows about a repository it got
// from here, so that one set of grammars and one graph shape serve every
// client.
package core

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

// Envelope is the shape every core route answers in.
type envelope[T any] struct {
	Status  string `json:"status"`
	Message string `json:"message"`
	Data    T      `json:"data"`
}

// Repository is one repository registered on this machine.
type Repository struct {
	Name string `json:"name"`
	Path string `json:"path"`
}

// Node is one file, import or symbol in a repository's graph.
//
// Kind and Labels are not the same question. A Python file's kind is "python"
// and a Python function's kind is "function", so kind alone cannot tell them
// apart; labels can. A drawing colours by one and reads by the other.
type Node struct {
	ID     string   `json:"id"`
	Name   string   `json:"name"`
	Kind   string   `json:"kind"`
	Labels []string `json:"labels"`
	Path   string   `json:"path"`
}

// Link is a typed edge between two nodes.
type Link struct {
	Source string `json:"source"`
	Target string `json:"target"`
	Type   string `json:"type"`
}

// Graph is a whole scope, as the index drew it.
//
// Truncated says the scope was larger than the cap asked for, so what came back
// is a part of the repository and not the repository.
type Graph struct {
	Nodes     []Node `json:"nodes"`
	Links     []Link `json:"links"`
	Truncated bool   `json:"truncated"`
}

// Error is a non-2xx answer from the core, carrying what it said.
type Error struct {
	StatusCode int
	Detail     string
}

func (e *Error) Error() string {
	if e.Detail == "" {
		return fmt.Sprintf("sourceant core returned %d", e.StatusCode)
	}
	return fmt.Sprintf("sourceant core returned %d: %s", e.StatusCode, e.Detail)
}

// NotFound reports whether the core had no such repository registered.
func (e *Error) NotFound() bool { return e.StatusCode == http.StatusNotFound }

// Client talks to one core instance.
type Client struct {
	baseURL string
	http    *http.Client
}

// New builds a client for the core serving at baseURL.
func New(baseURL string, timeout time.Duration) *Client {
	return &Client{
		baseURL: strings.TrimRight(baseURL, "/"),
		http:    &http.Client{Timeout: timeout},
	}
}

// BaseURL is the core this client talks to.
func (c *Client) BaseURL() string { return c.baseURL }

// Healthy reports whether the core is up and answering.
func (c *Client) Healthy(ctx context.Context) bool {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL+"/health", nil)
	if err != nil {
		return false
	}
	resp, err := c.http.Do(req)
	if err != nil {
		return false
	}
	defer func() { _ = resp.Body.Close() }()
	_, _ = io.Copy(io.Discard, resp.Body)
	return resp.StatusCode >= 200 && resp.StatusCode < 300
}

// Repositories lists what has been registered on this machine.
func (c *Client) Repositories(ctx context.Context) ([]Repository, error) {
	return get[[]Repository](ctx, c, "/api/code/repositories", nil)
}

// GraphOptions narrows what a drawing covers.
type GraphOptions struct {
	PathPrefix   string
	IncludeTests bool
	NodeLimit    int
}

// Graph reads one repository's whole scope.
func (c *Client) Graph(ctx context.Context, repository string, opts GraphOptions) (Graph, error) {
	query := url.Values{"repository": {repository}}
	if opts.PathPrefix != "" {
		query.Set("path_prefix", opts.PathPrefix)
	}
	if opts.IncludeTests {
		query.Set("include_tests", "true")
	}
	if opts.NodeLimit > 0 {
		query.Set("node_limit", strconv.Itoa(opts.NodeLimit))
	}
	return get[Graph](ctx, c, "/api/code/graph", query)
}

// Register covers one more directory, so the next index run reads it too.
func (c *Client) Register(ctx context.Context, path, name string) (Repository, error) {
	return send[Repository](ctx, c, http.MethodPost, "/api/code/repositories", nil, map[string]string{
		"path": path,
		"name": name,
	})
}

// Forget stops covering a directory. What was already indexed is left alone.
func (c *Client) Forget(ctx context.Context, path string) error {
	_, err := send[map[string]any](ctx, c, http.MethodDelete, "/api/code/repositories",
		url.Values{"path": {path}}, nil)
	return err
}

// Indexed is what one repository's index run read.
type Indexed struct {
	Repository string `json:"repository"`
	Files      int    `json:"indexed"`
	Unchanged  int    `json:"unchanged"`
	Removed    int    `json:"removed"`
	Skipped    int    `json:"skipped"`
}

// Index reads repositories into the graph, one or all of them.
//
// The core answers when the reading is done, so this takes as long as the
// repository is large. The caller's context is what bounds it.
func (c *Client) Index(ctx context.Context, repository string, everything bool) ([]Indexed, error) {
	return send[[]Indexed](ctx, c, http.MethodPost, "/api/code/index", nil, map[string]any{
		"repository": repository,
		"everything": everything,
		"update":     true,
	})
}

// Knowledge is one thing recorded about a repository.
type Knowledge struct {
	ID         string         `json:"id"`
	Kind       string         `json:"kind"`
	Status     string         `json:"status"`
	Summary    string         `json:"summary"`
	Properties map[string]any `json:"properties"`
}

// KnowledgePage is what a search answered.
type KnowledgePage struct {
	Items   []Knowledge `json:"items"`
	Total   int         `json:"total"`
	HasMore bool        `json:"has_more"`
}

// Knowledge reads what is recorded about one repository.
func (c *Client) Knowledge(ctx context.Context, repository string, limit, offset int) (KnowledgePage, error) {
	query := url.Values{"repository": {repository}}
	if limit > 0 {
		query.Set("limit", strconv.Itoa(limit))
	}
	if offset > 0 {
		query.Set("offset", strconv.Itoa(offset))
	}
	return get[KnowledgePage](ctx, c, "/api/knowledge", query)
}

// RecordKnowledge writes something down about a repository.
func (c *Client) RecordKnowledge(ctx context.Context, repository string, item Knowledge) (Knowledge, error) {
	return send[Knowledge](ctx, c, http.MethodPut, "/api/knowledge", nil, map[string]any{
		"repository": repository,
		"id":         item.ID,
		"kind":       item.Kind,
		"status":     item.Status,
		"summary":    item.Summary,
		"properties": item.Properties,
	})
}

// ForgetKnowledge removes something recorded.
func (c *Client) ForgetKnowledge(ctx context.Context, repository, id string) error {
	_, err := send[map[string]any](ctx, c, http.MethodDelete, "/api/knowledge",
		url.Values{"repository": {repository}, "id": {id}}, nil)
	return err
}

func send[T any](ctx context.Context, c *Client, method, path string, query url.Values, payload any) (T, error) {
	var zero T
	target := c.baseURL + path
	if len(query) > 0 {
		target += "?" + query.Encode()
	}

	var body io.Reader
	if payload != nil {
		encoded, err := json.Marshal(payload)
		if err != nil {
			return zero, err
		}
		body = bytes.NewReader(encoded)
	}

	req, err := http.NewRequestWithContext(ctx, method, target, body)
	if err != nil {
		return zero, err
	}
	req.Header.Set("Accept", "application/json")
	if payload != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	resp, err := c.http.Do(req)
	if err != nil {
		return zero, err
	}
	defer func() { _ = resp.Body.Close() }()

	answered, err := io.ReadAll(resp.Body)
	if err != nil {
		return zero, err
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return zero, &Error{StatusCode: resp.StatusCode, Detail: detail(answered)}
	}

	var parsed envelope[T]
	if err := json.Unmarshal(answered, &parsed); err != nil {
		return zero, fmt.Errorf("sourceant core answered %s with something other than JSON: %w", path, err)
	}
	return parsed.Data, nil
}

func get[T any](ctx context.Context, c *Client, path string, query url.Values) (T, error) {
	var zero T
	target := c.baseURL + path
	if len(query) > 0 {
		target += "?" + query.Encode()
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, target, nil)
	if err != nil {
		return zero, err
	}
	req.Header.Set("Accept", "application/json")

	resp, err := c.http.Do(req)
	if err != nil {
		return zero, err
	}
	defer func() { _ = resp.Body.Close() }()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return zero, err
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return zero, &Error{StatusCode: resp.StatusCode, Detail: detail(body)}
	}

	var parsed envelope[T]
	if err := json.Unmarshal(body, &parsed); err != nil {
		return zero, fmt.Errorf("sourceant core answered %s with something other than JSON: %w", path, err)
	}
	return parsed.Data, nil
}

// detail pulls the reason out of an error body, falling back to the body itself.
func detail(body []byte) string {
	var parsed struct {
		Detail string `json:"detail"`
		Error  string `json:"error"`
	}
	if err := json.Unmarshal(body, &parsed); err == nil {
		if parsed.Detail != "" {
			return parsed.Detail
		}
		if parsed.Error != "" {
			return parsed.Error
		}
	}
	return strings.TrimSpace(string(body))
}
