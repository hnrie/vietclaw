package web

import (
	"net/http"
	"os"
	"strings"

	"vietclaw/internal/app"
)

const defaultRecentLogLines = 80

func handleRecentLogs(application *app.App) http.HandlerFunc {
	return func(w http.ResponseWriter, _ *http.Request) {
		lines, err := recentLines(application.LogFile, defaultRecentLogLines)
		if err != nil {
			application.Logger.Printf("read recent logs: %v", err)
			writeJSON(w, http.StatusOK, []string{})
			return
		}
		writeJSON(w, http.StatusOK, lines)
	}
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
