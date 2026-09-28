package core

import (
	"context"
	"encoding/json"
	"net/http"
	"net/url"
	"strconv"
)

func (c *Client) Architecture(ctx context.Context, repository string, depth int, includeTests bool) (json.RawMessage, error) {
	return get[json.RawMessage](ctx, c, "/api/code/architecture", url.Values{
		"repository": {repository}, "depth": {strconv.Itoa(depth)}, "include_tests": {strconv.FormatBool(includeTests)},
	})
}

func (c *Client) CompareArchitecture(ctx context.Context, baseline json.RawMessage) (json.RawMessage, error) {
	return send[json.RawMessage](ctx, c, http.MethodPost, "/api/code/architecture/compare", nil, baseline)
}
