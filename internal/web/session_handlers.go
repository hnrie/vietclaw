package web

import (
	"net/http"

	"vietclaw/internal/app"
)

func handleSessions(application *app.App) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		sessions, err := application.Agent.Sessions(r.Context())
		if err != nil {
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}
		writeJSON(w, http.StatusOK, sessions)
	}
}

func handleSessionDetail(application *app.App) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		detail, err := application.Agent.SessionMessages(r.Context(), r.PathValue("id"))
		if err != nil {
			writeError(w, http.StatusNotFound, "session not found")
			return
		}
		writeJSON(w, http.StatusOK, detail)
	}
}
