package server

import (
	"io/fs"
	"net/http"
	"strings"
)

// spaHandler serves static files from fsys.
// Any path that doesn't correspond to a real file falls back to index.html
// so that client-side routes (e.g. /about, /settings) work after a hard refresh.
func spaHandler(fsys fs.FS) http.Handler {
	fileServer := http.FileServer(http.FS(fsys))

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Strip the leading slash; fs.FS paths are rooted without it.
		path := strings.TrimPrefix(r.URL.Path, "/")
		if path == "" {
			path = "."
		}

		if _, err := fs.Stat(fsys, path); err != nil {
			// File not found — let the SPA handle this route.
			r.URL.Path = "/"
		}

		fileServer.ServeHTTP(w, r)
	})
}
