// Package ui serves the local graph view.
//
// The assets are embedded rather than fetched, so the view works on a laptop
// with no network and cannot drift from the binary serving it.
package ui

import (
	"embed"
	"io/fs"
	"net/http"
	"path"
	"strings"
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
	return noStore(spa(files, http.FileServer(http.FS(files))))
}

// spa answers a path the page owns with the page.
//
// Routes are real paths rather than fragments after a "#", so a link somebody
// is handed can be opened, bookmarked and pasted like any other URL. The
// browser then asks this server for "/reviews/abc123", which is not a file.
// Anything that is not a file is the application, and the application works
// out what to draw from the path once it is running.
func spa(files fs.FS, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet && isRoute(files, r.URL.Path) {
			r = r.Clone(r.Context())
			r.URL.Path = "/"
		}
		next.ServeHTTP(w, r)
	})
}

// isRoute is whether a path is the application's rather than a file's.
//
// An asset that is genuinely missing keeps its 404. Answering it with the page
// would hand a script tag an HTML document, and the failure would surface
// somewhere further away than the missing file.
func isRoute(files fs.FS, name string) bool {
	clean := strings.TrimPrefix(path.Clean(name), "/")
	if clean == "" || clean == "." {
		return false
	}
	if strings.HasPrefix(clean, "assets/") || path.Ext(clean) != "" {
		return false
	}
	stat, err := fs.Stat(files, clean)
	return err != nil || stat.IsDir()
}

// noStore keeps a browser from holding on to a view the next agent replaces.
func noStore(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Cache-Control", "no-store")
		next.ServeHTTP(w, r)
	})
}
