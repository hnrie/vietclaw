package web

import (
	"io/fs"
	"mime"
	"net/http"
	"path"
	"strings"
	"time"

	"vietclaw/internal/app"
)

const (
	distRoot      = "dist"
	indexFile     = "index.html"
	apiPathPrefix = "/api/"
)

func handleStatic(application *app.App) http.HandlerFunc {
	dist, err := fs.Sub(webDist, distRoot)
	if err != nil {
		application.Logger.Printf("load embedded web dist: %v", err)
	}
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet && r.Method != http.MethodHead {
			http.NotFound(w, r)
			return
		}
		if strings.HasPrefix(r.URL.Path, apiPathPrefix) {
			http.NotFound(w, r)
			return
		}

		name := strings.TrimPrefix(path.Clean(r.URL.Path), "/")
		if name == "." || name == "" {
			name = indexFile
		}
		if file, err := dist.Open(name); err == nil {
			defer file.Close()
			if info, statErr := file.Stat(); statErr == nil && !info.IsDir() {
				serveEmbeddedFile(w, r, name, file, info.ModTime())
				return
			}
		}
		file, err := dist.Open(indexFile)
		if err != nil {
			http.Error(w, "web UI is not available", http.StatusServiceUnavailable)
			return
		}
		defer file.Close()
		serveEmbeddedFile(w, r, indexFile, file, time.Now())
	}
}

func serveEmbeddedFile(w http.ResponseWriter, r *http.Request, name string, file fs.File, modTime time.Time) {
	if strings.HasPrefix(name, "_nuxt/") || strings.HasPrefix(name, "assets/") {
		w.Header().Set("Cache-Control", "public, max-age=31536000, immutable")
	} else {
		w.Header().Set("Cache-Control", "no-cache")
	}
	if contentType := mime.TypeByExtension(path.Ext(name)); contentType != "" {
		w.Header().Set("Content-Type", contentType)
	}
	reader, ok := file.(interface {
		Read([]byte) (int, error)
		Seek(int64, int) (int64, error)
	})
	if !ok {
		http.Error(w, "embedded file is not seekable", http.StatusInternalServerError)
		return
	}
	http.ServeContent(w, r, name, modTime, reader)
}
