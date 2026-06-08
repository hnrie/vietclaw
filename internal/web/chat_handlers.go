package web

import (
	"encoding/json"
	"fmt"
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

func handleAPIChatStream(application *app.App) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req agent.ChatRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeError(w, http.StatusBadRequest, "invalid json")
			return
		}

		flusher, ok := w.(http.Flusher)
		if !ok {
			writeError(w, http.StatusInternalServerError, "streaming not supported")
			return
		}

		w.Header().Set("Content-Type", "text/event-stream")
		w.Header().Set("Cache-Control", "no-cache")
		w.Header().Set("Connection", "keep-alive")

		ch, err := application.Agent.ChatStream(r.Context(), req)
		if err != nil {
			writeSSEJSON(w, "error", map[string]string{"error": err.Error()})
			flusher.Flush()
			return
		}

		for chunk := range ch {
			if chunk.Error != "" {
				writeSSEJSON(w, "error", map[string]string{"error": chunk.Error})
				flusher.Flush()
				return
			}
			if chunk.Done {
				fmt.Fprintf(w, "event: done\ndata: [DONE]\n\n")
				flusher.Flush()
				break
			}
			if chunk.Text != "" {
				writeSSEJSON(w, "", map[string]string{"text": chunk.Text})
				flusher.Flush()
			}
		}
	}
}

func writeSSEJSON(w http.ResponseWriter, event string, value any) {
	payload, _ := json.Marshal(value)
	if event != "" {
		fmt.Fprintf(w, "event: %s\n", event)
	}
	fmt.Fprintf(w, "data: %s\n\n", string(payload))
}
