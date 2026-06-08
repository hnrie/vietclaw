package web

import (
	"net/http"
	"time"

	"vietclaw/internal/app"
	"vietclaw/internal/db"
)

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
