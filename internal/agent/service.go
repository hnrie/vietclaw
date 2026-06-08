package agent

import (
	"database/sql"

	"vietclaw/internal/config"
	contextbuilder "vietclaw/internal/context"
	"vietclaw/internal/i18n"
	"vietclaw/internal/memory"
	"vietclaw/internal/providers"
	"vietclaw/internal/router"
	"vietclaw/internal/tools"
)

type Service struct {
	cfg     config.Config
	db      *sql.DB
	mem     *memory.Store
	router  *router.ModelRouter
	context *contextbuilder.Builder
	tools   *tools.ToolRegistry
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
		tools:   tools.NewRegistry(cfg),
	}
}

func (s *Service) Memory() *memory.Store {
	return s.mem
}

func (s *Service) Language() string {
	return string(i18n.Normalize(s.cfg.Agent.Language))
}

func (s *Service) text(id i18n.MessageID, args ...any) string {
	return i18n.T(s.cfg.Agent.Language, id, args...)
}

func (s *Service) Router() *router.ModelRouter {
	return s.router
}

