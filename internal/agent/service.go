package agent

import (
	"database/sql"

	"vietclaw/internal/config"
	contextbuilder "vietclaw/internal/context"
	"vietclaw/internal/memory"
	"vietclaw/internal/providers"
	"vietclaw/internal/router"
)

type Service struct {
	cfg     config.Config
	db      *sql.DB
	mem     *memory.Store
	router  *router.ModelRouter
	context *contextbuilder.Builder
}

func NewService(cfg config.Config, db *sql.DB) *Service {
	mem := memory.NewStore(db)
	providerList := providers.Enabled(cfg.Providers)
	return &Service{
		cfg:     cfg,
		db:      db,
		mem:     mem,
		router:  router.NewModelRouter(cfg, db, providerList),
		context: contextbuilder.New(cfg, db, mem),
	}
}

func (s *Service) Memory() *memory.Store {
	return s.mem
}
