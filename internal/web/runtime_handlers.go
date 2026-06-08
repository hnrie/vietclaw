package web

import (
	"net/http"
	"os"

	"vietclaw/internal/app"
	"vietclaw/internal/channels"
	"vietclaw/internal/providers"
	"vietclaw/internal/router"
)

func handleCostsToday(application *app.App) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, map[string]any{"total_cost_usd": router.TodayCost(r.Context(), application.DB)})
	}
}

func handleBudget(application *app.App) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, map[string]any{
			"total_cost_usd":             router.TodayCost(r.Context(), application.DB),
			"daily_usd_limit":            application.Config.Budget.DailyUSDLimit,
			"require_approval_above_usd": application.Config.Budget.RequireApprovalAboveUSD,
			"cheap_first":                application.Config.Router.CheapFirst,
			"allow_escalation":           application.Config.Router.AllowEscalation,
		})
	}
}

func handleProviders(application *app.App) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, providers.Redact(application.Config.Providers))
	}
}

func handleChannels(application *app.App) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if application.Channels != nil {
			writeJSON(w, http.StatusOK, application.Channels.Statuses())
			return
		}
		writeJSON(w, http.StatusOK, channels.StatusFromConfig(application.Config))
	}
}

func handleDiscordTest(application *app.App) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, channelEnvStatus("discord", application.Config.Channels.Discord.Enabled, application.Config.Channels.Discord.TokenEnv))
	}
}

func handleTelegramTest(application *app.App) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, channelEnvStatus("telegram", application.Config.Channels.Telegram.Enabled, application.Config.Channels.Telegram.TokenEnv))
	}
}

func channelEnvStatus(name string, enabled bool, tokenEnv string) map[string]any {
	_, ok := os.LookupEnv(tokenEnv)
	return map[string]any{
		"name":      name,
		"enabled":   enabled,
		"token_env": tokenEnv,
		"env_found": ok,
	}
}
