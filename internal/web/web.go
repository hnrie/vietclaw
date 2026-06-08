package web

import (
	"encoding/json"
	"net/http"
	"os"
	"strings"
	"time"

	"vietclaw/internal/app"
	"vietclaw/internal/db"
	webassets "vietclaw/web"
)

func NewRouter(application *app.App) http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /", handleIndex(application))
	mux.HandleFunc("GET /health", handleHealth)
	mux.HandleFunc("GET /status", handleStatus(application))
	mux.HandleFunc("GET /logs/recent", handleRecentLogs(application))
	return mux
}

func handleIndex(application *app.App) http.HandlerFunc {
	tmpl := webassets.IndexTemplate()
	return func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/" {
			http.NotFound(w, r)
			return
		}

		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		data := map[string]any{
			"Version": application.Version.Version,
			"Uptime":  time.Since(application.StartTime).Round(time.Second).String(),
		}
		if err := tmpl.Execute(w, data); err != nil {
			application.Logger.Printf("render index: %v", err)
		}
	}
}

func handleHealth(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]bool{"ok": true})
}

func handleStatus(application *app.App) http.HandlerFunc {
	return func(w http.ResponseWriter, _ *http.Request) {
		writeJSON(w, http.StatusOK, map[string]any{
			"version":              application.Version.Version,
			"commit":               application.Version.Commit,
			"uptime":               time.Since(application.StartTime).Round(time.Second).String(),
			"db_ok":                db.Check(application.DB),
			"mode":                 application.Config.Runtime.Mode,
			"max_concurrent_tasks": application.Config.Runtime.MaxConcurrentTasks,
		})
	}
}

func handleRecentLogs(application *app.App) http.HandlerFunc {
	return func(w http.ResponseWriter, _ *http.Request) {
		lines, err := recentLines(application.LogFile, 80)
		if err != nil {
			application.Logger.Printf("read recent logs: %v", err)
			writeJSON(w, http.StatusOK, []string{})
			return
		}
		writeJSON(w, http.StatusOK, lines)
	}
}

func writeJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}

func recentLines(path string, maxLines int) ([]string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return []string{}, nil
		}
		return nil, err
	}

	text := strings.TrimRight(string(data), "\r\n")
	if text == "" {
		return []string{}, nil
	}

	lines := strings.Split(text, "\n")
	if len(lines) <= maxLines {
		return lines, nil
	}
	return lines[len(lines)-maxLines:], nil
}
