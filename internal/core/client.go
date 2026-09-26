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
//
// IndexedAt is empty until it has been read, which is not the same as nothing
// having changed since, and Reading says a read is under way now.
type Repository struct {
	Name      string `json:"name"`
	Path      string `json:"path"`
	IndexedAt string `json:"indexed_at"`
	Reading   bool   `json:"reading"`
}

// Node is one file, import or symbol in a repository's graph.
//
// Kind says what the thing is and never what it is written in: a file is a file
// whatever its language. Degree is how many lines meet here, which is what
// sizes it, and Community is which part of the repository it belongs to, which
// is what colours it.
type Node struct {
	ID        string   `json:"id"`
	Name      string   `json:"name"`
	Kind      string   `json:"kind"`
	Language  string   `json:"language,omitempty"`
	Path      string   `json:"path"`
	Labels    []string `json:"labels"`
	Degree    int      `json:"degree"`
	Community *int     `json:"community"`
}

// Community is one part of a code graph: symbols more connected to each other
// than to the rest, named after where they live or what they are built around.
type Community struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
	Size int    `json:"size"`
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
	Nodes       []Node      `json:"nodes"`
	Links       []Link      `json:"links"`
	Communities []Community `json:"communities"`
	Truncated   bool        `json:"truncated"`
	// Focus is the node it was walked out from, empty if it drew everything.
	Focus string `json:"focus,omitempty"`
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

// Patience is how long an ordinary call may take. Reads and writes against a
// local index answer in milliseconds; anything near this is a core in trouble.
const Patience = 30 * time.Second

// Working is how long a call that does real work may take. Reading a repository
// of ten thousand files, or asking a model about five rules one at a time, is
// minutes rather than seconds, and cutting it off at the ordinary deadline
// throws away work that was going to succeed.
const Working = 15 * time.Minute

// Client talks to one core instance.
type Client struct {
	baseURL  string
	http     *http.Client
	patience time.Duration
}

// New builds a client for the core serving at baseURL.
//
// The deadline is per call rather than on the client itself, because a client
// deadline caps every call at the shortest one any call needs.
func New(baseURL string, timeout time.Duration) *Client {
	if timeout <= 0 {
		timeout = Patience
	}
	return &Client{
		baseURL:  strings.TrimRight(baseURL, "/"),
		http:     &http.Client{},
		patience: timeout,
	}
}

// waiting gives a call a deadline, unless it already has one.
//
// A deadline already on the context was set by whoever knows what this call is
// doing, so it wins: applying the ordinary one on top would cut a fifteen
// minute review off after thirty seconds.
func waiting(ctx context.Context, limit time.Duration) (context.Context, context.CancelFunc) {
	if _, ok := ctx.Deadline(); ok {
		return ctx, func() {}
	}
	return context.WithTimeout(ctx, limit)
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

// Worth is one file where recent change has landed on something the rest of
// the code leans on.
type Worth struct {
	Path       string `json:"path"`
	Dependants int    `json:"dependants"`
	Changes    int    `json:"changes"`
}

// Attention is where a person's time goes furthest in a repository.
type Attention struct {
	Files []Worth `json:"files"`
	// The window the change counts cover, so a screen need not invent one.
	Since string `json:"since"`
}

// Attention is the files where recent change meets a central position.
//
// Either fact alone says little: something half the codebase imports and
// nobody has touched is settled, and something nothing imports that changes
// daily is a scratch pad. It is the overlap that is worth somebody's time.
func (c *Client) Attention(ctx context.Context, repository string) (Attention, error) {
	return get[Attention](ctx, c, "/api/code/attention", url.Values{
		"repository": {repository},
	})
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
// Update reads only what changed since last time, which is what a watcher
// wants. Somebody who asked for this in as many words usually means read it
// again: how a file is read changes with the indexer, and an update pass sees
// an unchanged file and skips it.
//
// The core answers when the reading is done, so this takes as long as the
// repository is large. The caller's context is what bounds it.
func (c *Client) Index(ctx context.Context, repository string, everything, update bool) ([]Indexed, error) {
	ctx, done := waiting(ctx, Working)
	defer done()
	return send[[]Indexed](ctx, c, http.MethodPost, "/api/code/index", nil, map[string]any{
		"repository": repository,
		"everything": everything,
		"update":     update,
	})
}

// Knowledge is one thing recorded about a repository.
type Knowledge struct {
	ID     string `json:"id"`
	Kind   string `json:"kind"`
	Status string `json:"status"`
	// Which repository it was recorded against. Only answered when the search
	// was not narrowed to one, where it is the thing a reader cannot infer.
	Repository string         `json:"repository,omitempty"`
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

// Seed is one thing a repository states, read off a file rather than judged.
type Seed struct {
	Knowledge
	Source string `json:"source"`
}

// Seeded is what reading a repository's own words found.
type Seeded struct {
	Found    []Seed `json:"found"`
	Recorded int    `json:"recorded"`
}

// Initialize records what a repository already states about itself.
//
// Asking without recording is the safe half, so a person can see what would be
// written before any of it is. Asking a model as well finds what nobody wrote
// down, and costs whatever the machine's own model costs.
func (c *Client) Initialize(ctx context.Context, repository string, dryRun, useModel bool) (Seeded, error) {
	if useModel {
		var done context.CancelFunc
		ctx, done = waiting(ctx, Working)
		defer done()
	}
	return send[Seeded](ctx, c, http.MethodPost, "/api/knowledge/initialize", nil, map[string]any{
		"repository": repository,
		"dry_run":    dryRun,
		"use_model":  useModel,
	})
}

// Skill is one thing a team wrote down about how work here is done.
//
// Paths, Reviews and Automatic are what the author stated in the skill's own
// frontmatter: which files it is about, whether it belongs in a review, and
// whether anything but a person may start it. Reviews is null where nobody
// said, which is most of them and is not the same as saying no.
type Skill struct {
	ID          string   `json:"id"`
	Name        string   `json:"name"`
	Description string   `json:"description"`
	Origin      string   `json:"origin"`
	Path        string   `json:"path"`
	Paths       []string `json:"paths"`
	Reviews     *bool    `json:"reviews"`
	Automatic   bool     `json:"automatic"`
	Body        string   `json:"body,omitempty"`
}

// SkillPage is the skills on hand.
type SkillPage struct {
	Skills []Skill `json:"skills"`
	Total  int     `json:"total"`
}

// Skills is everything this machine and this repository hold.
//
// All of them: a person with a folder per coding agent easily has a hundred,
// and a screen that silently showed the first fifty would be lying about what
// a review had to choose from.
func (c *Client) Skills(ctx context.Context, repository string) (SkillPage, error) {
	return get[SkillPage](ctx, c, "/api/skills", url.Values{
		"repository": {repository},
		"limit":      {"500"},
	})
}

// Skill is one rule in full, so a person can read what a check was made against.
func (c *Client) Skill(ctx context.Context, id, repository string) (Skill, error) {
	return get[Skill](ctx, c, "/api/skills/"+id, url.Values{"repository": {repository}})
}

// Stated is a skill somebody is writing down, and where it belongs.
//
// Scope is "repository" for something about one project, which the team then
// gets by pulling, or "machine" for something somebody wants everywhere.
type Stated struct {
	ID          string   `json:"id"`
	Repository  string   `json:"repository"`
	Scope       string   `json:"scope"`
	Name        string   `json:"name"`
	Description string   `json:"description"`
	Body        string   `json:"body"`
	Paths       []string `json:"paths"`
	Reviews     *bool    `json:"reviews"`
}

// RecordSkill writes a skill down, in a repository or on this machine.
//
// Only the places this product owns are written. What somebody keeps in the
// folders named after a coding agent is that agent's, and core refuses to
// write there.
func (c *Client) RecordSkill(ctx context.Context, stated Stated) (Skill, error) {
	if stated.Scope == "" {
		stated.Scope = "repository"
	}
	if stated.Paths == nil {
		stated.Paths = []string{}
	}
	return send[Skill](ctx, c, http.MethodPut, "/api/skills", nil, stated)
}

// ForgetSkill removes a skill written here.
func (c *Client) ForgetSkill(ctx context.Context, repository, scope, id string) error {
	if scope == "" {
		scope = "repository"
	}
	_, err := send[map[string]any](ctx, c, http.MethodDelete, "/api/skills",
		url.Values{"repository": {repository}, "scope": {scope}, "id": {id}}, nil)
	return err
}

// Finding is one thing a rule says is wrong with a change.
type Finding struct {
	Detail   string `json:"detail"`
	Severity string `json:"severity"`
	Path     string `json:"path"`
	Line     *int   `json:"line"`
}

// Verdict is what one rule made of a change.
type Verdict struct {
	Skill    string    `json:"skill"`
	Passed   bool      `json:"passed"`
	Note     string    `json:"note"`
	Findings []Finding `json:"findings"`
}

// ChangedFile is one file a checkout's work touches, and what changed in it.
//
// The patch travels with the file rather than as one diff for the whole
// change: a page shows somebody the file they are looking at, and a list of
// names is not a review.
type ChangedFile struct {
	Path   string `json:"path"`
	Change string `json:"change"`
	Patch  string `json:"patch"`
}

// Recorded is one thing known about the repository being reviewed.
type Recorded struct {
	ID      string `json:"id"`
	Kind    string `json:"kind"`
	Summary string `json:"summary"`
}

// Where is the checkout a review read, and what it was compared against.
//
// A person with a worktree open somewhere else is otherwise left wondering
// whose work they are looking at.
type Where struct {
	Path    string `json:"path"`
	Branch  string `json:"branch"`
	Against string `json:"against"`
	Base    string `json:"base"`
	Commits int    `json:"commits"`
}

// Suggestion is one thing to change, and the code to put there.
type Suggestion struct {
	Path      string `json:"path"`
	StartLine int    `json:"start_line"`
	EndLine   int    `json:"end_line"`
	Side      string `json:"side"`
	Comment   string `json:"comment"`
	Category  string `json:"category"`
	// Both sides. Without what it replaces, a suggestion draws as an addition
	// out of nowhere.
	ExistingCode  string `json:"existing_code"`
	SuggestedCode string `json:"suggested_code"`
}

// Summary is the review in the order a person reads it.
type Summary struct {
	Overview         string   `json:"overview"`
	KeyImprovements  []string `json:"key_improvements"`
	MinorSuggestions []string `json:"minor_suggestions"`
	CriticalIssues   []string `json:"critical_issues"`
}

// Read is the review proper: the same one the hosted path gives a pull
// request, from the same generator.
type Read struct {
	Verdict     string            `json:"verdict"`
	Summary     Summary           `json:"summary"`
	Suggestions []Suggestion      `json:"suggestions"`
	Notes       map[string]string `json:"notes"`
}

// Commit is one commit the branch has that the branch it left does not.
type Commit struct {
	SHA     string `json:"sha"`
	Author  string `json:"author"`
	At      string `json:"at"`
	Subject string `json:"subject"`
	Body    string `json:"body"`
}

// Review is whether a checkout's work is ready to be proposed to anyone.
//
// Every field the core answers with has to be named here. This is decoded into
// and re-encoded on the way out, so anything missing is dropped in silence.
type Review struct {
	Ready     bool          `json:"ready"`
	Note      string        `json:"note"`
	Base      string        `json:"base"`
	Where     Where         `json:"where"`
	Changed   []ChangedFile `json:"changed"`
	Commits   []Commit      `json:"commits"`
	Skills    []Skill       `json:"skills"`
	Knowledge []Recorded    `json:"knowledge"`
	Verdicts  []Verdict     `json:"verdicts"`
	// The review itself, as opposed to what the skills made of it.
	Read Read `json:"review"`
}

// Ask is what to review and how.
type Ask struct {
	Repository  string   `json:"repository"`
	Against     string   `json:"against"`
	Title       string   `json:"title"`
	Description string   `json:"description"`
	Skills      []string `json:"skills"`
	UseModel    bool     `json:"use_model"`
}

// Reading is one review, whether it has finished or not.
//
// Kept by the core rather than held here, because the thing that asks for a
// review is often not the thing that reads it: an agent runs one over MCP and
// hands somebody a link, and the link has to still work later.
type Reading struct {
	ID         string `json:"id"`
	Repository string `json:"repository"`
	Status     string `json:"status"`
	Title      string `json:"title"`
	Error      string `json:"error"`
	Started    string `json:"started"`
	Finished   string `json:"finished"`
	Review     Review `json:"review"`
	// Where to send somebody who was handed this by an agent.
	Path string `json:"path"`
}

// Review asks for a review and answers with where to find it.
//
// Nothing here reaches a forge: the work being judged has not been proposed to
// anyone yet, which is the point of judging it now.
func (c *Client) Review(ctx context.Context, ask Ask) (Reading, error) {
	if ask.Skills == nil {
		ask.Skills = []string{}
	}
	return send[Reading](ctx, c, http.MethodPost, "/api/local/reviews", nil, ask)
}

// Reviewed is one review by name, however long ago it ran.
func (c *Client) Reviewed(ctx context.Context, id string) (Reading, error) {
	return get[Reading](ctx, c, "/api/local/reviews/"+url.PathEscape(id), nil)
}

// Reviews is the last few, newest first, without their findings.
func (c *Client) Reviews(ctx context.Context, repository string) ([]Reading, error) {
	return get[[]Reading](ctx, c, "/api/local/reviews", url.Values{
		"repository": {repository},
	})
}

// Setting is one thing configurable on this machine.
//
// A credential answers whether it is set rather than what it is: a screen needs
// the first and nothing needs the second.
type Setting struct {
	Key         string   `json:"key"`
	Label       string   `json:"label"`
	Description string   `json:"description"`
	Type        string   `json:"type"`
	Value       any      `json:"value"`
	Default     any      `json:"default"`
	Choices     []string `json:"choices"`
	Group       string   `json:"group"`
	Secret      bool     `json:"secret"`
	// Listed is several of something rather than one thing, kept one to a
	// line, so a screen draws it as a list rather than as a box of text.
	Listed bool  `json:"listed"`
	IsSet  *bool `json:"is_set"`
}

// Settings is everything configurable on this machine.
func (c *Client) Settings(ctx context.Context) ([]Setting, error) {
	return get[[]Setting](ctx, c, "/api/local/settings", nil)
}

// SetSetting gives one setting a value on this machine.
func (c *Client) SetSetting(ctx context.Context, key string, value any) (Setting, error) {
	return send[Setting](ctx, c, http.MethodPut, "/api/local/settings/"+url.PathEscape(key),
		nil, map[string]any{"value": value})
}

// ResetSetting puts one setting back to what it would be if nobody had touched it.
func (c *Client) ResetSetting(ctx context.Context, key string) (Setting, error) {
	return send[Setting](ctx, c, http.MethodDelete,
		"/api/local/settings/"+url.PathEscape(key), nil, nil)
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

	ctx, done := waiting(ctx, c.patience)
	defer done()

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
	ctx, done := waiting(ctx, c.patience)
	defer done()

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
