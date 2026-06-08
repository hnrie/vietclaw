package app

import (
	"database/sql"
	"log"
	"time"

	"vietclaw/internal/agent"
	"vietclaw/internal/channels"
	"vietclaw/internal/config"
	"vietclaw/internal/version"
)

type App struct {
	Config    config.Config
	DB        *sql.DB
	Logger    *log.Logger
	StartTime time.Time
	Version   version.Info
	DataDir   string
	LogFile   string
	Agent     *agent.Service
	Channels  *channels.Manager
}
