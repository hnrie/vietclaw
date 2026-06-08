package web

import (
	"encoding/json"
	"net/http"

	"vietclaw/internal/app"
	"vietclaw/internal/memory"
)

const (
	defaultMemoryListLimit   = 100
	defaultMemorySearchLimit = 50
)

func handleMemoryList(application *app.App) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		records, err := application.Agent.Memory().List(r.Context(), r.URL.Query().Get("scope"), defaultMemoryListLimit)
		if err != nil {
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}
		writeJSON(w, http.StatusOK, records)
	}
}

func handleMemoryAdd(application *app.App) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			Scope      string `json:"scope"`
			Kind       string `json:"kind"`
			Content    string `json:"content"`
			Confidence string `json:"confidence"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeError(w, http.StatusBadRequest, "invalid json")
			return
		}

		embedder := application.Agent.Router().SelectDefaultEmbedder()
		var embedding []float32
		if embedder != nil {
			embedding, _ = embedder.Embed(r.Context(), req.Content)
		}

		rec, err := application.Agent.Memory().Add(r.Context(), memory.Record{
			Scope:      req.Scope,
			Kind:       memory.Kind(req.Kind),
			Content:    req.Content,
			Confidence: memory.Confidence(req.Confidence),
			Embedding:  embedding,
		})
		if err != nil {
			writeError(w, http.StatusBadRequest, err.Error())
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"ok": true, "memory": rec})
	}
}

func handleMemorySearch(application *app.App) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		embedder := application.Agent.Router().SelectDefaultEmbedder()
		records, err := application.Agent.Memory().SearchHybrid(r.Context(), r.URL.Query().Get("scope"), r.URL.Query().Get("q"), defaultMemorySearchLimit, embedder)
		if err != nil {
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}
		writeJSON(w, http.StatusOK, records)
	}
}
