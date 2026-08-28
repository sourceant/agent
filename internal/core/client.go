// Package core reads the Python indexer's HTTP surface.
//
// The agent never parses code. Everything it knows about a repository it got
// from here, so that one set of grammars and one graph shape serve every
// client.
package core

import (
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
