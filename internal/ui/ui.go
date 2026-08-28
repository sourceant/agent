// Package ui serves the local graph view.
//
// The assets are embedded rather than fetched, so the view works on a laptop
// with no network and cannot drift from the binary serving it.
package ui

import (
	"embed"
	"io/fs"
	"net/http"
)

//go:embed assets
var assets embed.FS

// Handler serves the graph view at the root of whatever it is mounted on.
func Handler() http.Handler {
	files, err := fs.Sub(assets, "assets")
	if err != nil {
		// Only reachable if the embed directive and this path disagree, which
		// is a build-time mistake rather than anything a run can recover from.
		panic(err)
	}
	return noStore(http.FileServer(http.FS(files)))
}

// noStore keeps a browser from holding on to a view the next agent replaces.
func noStore(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Cache-Control", "no-store")
		next.ServeHTTP(w, r)
	})
}
