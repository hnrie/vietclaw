package web

import (
	"database/sql"
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"strings"

	"vietclaw/internal/app"
	"vietclaw/internal/harness"
)

func handleHarnessRuns(application *app.App) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		service := harness.New(application.Config, application.DB)
		switch r.Method {
		case http.MethodGet:
			limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
			runs, err := service.List(r.Context(), limit)
			if err != nil {
				writeError(w, http.StatusInternalServerError, err.Error())
				return
			}
			writeJSON(w, http.StatusOK, map[string]any{"runs": runs})
		case http.MethodPost:
			var req harness.CreateRequest
			if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
				writeError(w, http.StatusBadRequest, "invalid json")
				return
			}
			if strings.TrimSpace(req.Goal) == "" {
				writeError(w, http.StatusBadRequest, "goal is required")
				return
			}
			run, err := service.Create(r.Context(), req)
			if err != nil {
				writeError(w, http.StatusInternalServerError, err.Error())
				return
			}
			writeJSON(w, http.StatusOK, run)
		default:
			writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		}
	}
}

func handleHarnessRunDetail(application *app.App) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := r.PathValue("id")
		detail, err := harness.New(application.Config, application.DB).Detail(r.Context(), id)
		if errors.Is(err, sql.ErrNoRows) {
			writeError(w, http.StatusNotFound, "harness run not found")
			return
		}
		if err != nil {
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}
		writeJSON(w, http.StatusOK, detail)
	}
}

func handleHarnessRunStart(application *app.App) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		run, err := harness.New(application.Config, application.DB).Start(r.Context(), r.PathValue("id"))
		if errors.Is(err, sql.ErrNoRows) {
			writeError(w, http.StatusNotFound, "harness run not found")
			return
		}
		if err != nil {
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}
		writeJSON(w, http.StatusOK, run)
	}
}

func handleHarnessRunCancel(application *app.App) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		run, err := harness.New(application.Config, application.DB).Cancel(r.Context(), r.PathValue("id"))
		if errors.Is(err, sql.ErrNoRows) {
			writeError(w, http.StatusNotFound, "harness run not found")
			return
		}
		if err != nil {
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}
		writeJSON(w, http.StatusOK, run)
	}
}

func handleHarnessRunDiff(application *app.App) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		diff, err := harness.New(application.Config, application.DB).Diff(r.Context(), r.PathValue("id"))
		if errors.Is(err, sql.ErrNoRows) {
			writeError(w, http.StatusNotFound, "harness run not found")
			return
		}
		if err != nil {
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		_, _ = w.Write([]byte(diff))
	}
}

func handleHarnessRunCleanup(application *app.App) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		run, err := harness.New(application.Config, application.DB).Cleanup(r.Context(), r.PathValue("id"))
		if errors.Is(err, sql.ErrNoRows) {
			writeError(w, http.StatusNotFound, "harness run not found")
			return
		}
		if err != nil {
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}
		writeJSON(w, http.StatusOK, run)
	}
}
