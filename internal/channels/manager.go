package channels

import (
	"context"
	"log"
	"sync"

	"vietclaw/internal/config"
)

type Manager struct {
	adapters []Adapter
	logger   *log.Logger
	statuses map[string]Status
	mu       sync.Mutex
}

func NewManager(cfg config.Config, logger *log.Logger, adapters []Adapter) *Manager {
	statuses := map[string]Status{
		PlatformDiscord:  {Name: PlatformDiscord, Enabled: cfg.Channels.Discord.Enabled},
		PlatformTelegram: {Name: PlatformTelegram, Enabled: cfg.Channels.Telegram.Enabled},
	}
	return &Manager{logger: logger, adapters: adapters, statuses: statuses}
}

func (m *Manager) Start(ctx context.Context) {
	for _, adapter := range m.adapters {
		adapter := adapter
		m.setRunning(adapter.Name(), true, "")
		go func() {
			if m.logger != nil {
				m.logger.Printf("channel adapter starting name=%s", adapter.Name())
			}
			err := adapter.Start(ctx)
			if err != nil && ctx.Err() == nil {
				m.setRunning(adapter.Name(), false, err.Error())
				if m.logger != nil {
					m.logger.Printf("channel adapter failed name=%s err=%v", adapter.Name(), err)
				}
				return
			}
			m.setRunning(adapter.Name(), false, "")
			if m.logger != nil {
				m.logger.Printf("channel adapter stopped name=%s", adapter.Name())
			}
		}()
	}
}

func (m *Manager) Statuses() []Status {
	m.mu.Lock()
	defer m.mu.Unlock()
	return []Status{m.statuses[PlatformDiscord], m.statuses[PlatformTelegram]}
}

func (m *Manager) setRunning(name string, running bool, errText string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	status := m.statuses[name]
	status.Name = name
	status.Running = running
	status.Error = errText
	m.statuses[name] = status
}

func StatusFromConfig(cfg config.Config) []Status {
	return []Status{
		{Name: PlatformDiscord, Enabled: cfg.Channels.Discord.Enabled},
		{Name: PlatformTelegram, Enabled: cfg.Channels.Telegram.Enabled},
	}
}
