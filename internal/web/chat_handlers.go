package web

import (
	"encoding/json"
	"net/http"

	"vietclaw/internal/agent"
	"vietclaw/internal/app"
)

func handleAPIChat(application *app.App) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req agent.ChatRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeError(w, http.StatusBadRequest, "invalid json")
			return
		}
		resp, err := application.Agent.Chat(r.Context(), req)
		if err != nil {
			resp.OK = false
			if resp.Error == "" {
				resp.Error = err.Error()
			}
			writeJSON(w, http.StatusBadRequest, resp)
			return
		}
		writeJSON(w, http.StatusOK, resp)
	}
}
