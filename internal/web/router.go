package web

import (
	"net/http"

	"vietclaw/internal/app"
)

func NewRouter(application *app.App) http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /health", handleHealth)
	mux.HandleFunc("GET /status", handleStatus(application))
	mux.HandleFunc("GET /logs/recent", handleRecentLogs(application))
	mux.HandleFunc("GET /api/logs/recent", handleRecentLogs(application))
	mux.HandleFunc("POST /api/chat", handleAPIChat(application))
	mux.HandleFunc("GET /api/memory", handleMemoryList(application))
	mux.HandleFunc("POST /api/memory", handleMemoryAdd(application))
	mux.HandleFunc("GET /api/memory/search", handleMemorySearch(application))
	mux.HandleFunc("GET /api/sessions", handleSessions(application))
	mux.HandleFunc("GET /api/sessions/{id}", handleSessionDetail(application))
	mux.HandleFunc("GET /api/costs/today", handleCostsToday(application))
	mux.HandleFunc("GET /api/budget", handleBudget(application))
	mux.HandleFunc("GET /api/providers", handleProviders(application))
	mux.HandleFunc("GET /api/channels", handleChannels(application))
	mux.HandleFunc("POST /api/channels/discord/test", handleDiscordTest(application))
	mux.HandleFunc("POST /api/channels/telegram/test", handleTelegramTest(application))
	mux.HandleFunc("GET /", handleStatic(application))
	return mux
}
