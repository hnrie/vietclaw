package providers

import (
	"net/http"
	"time"

	"vietclaw/internal/config"
)

const defaultHTTPTimeout = 60 * time.Second

func New(cfg config.ProviderConfig) Provider {
	client := &http.Client{Timeout: defaultHTTPTimeout}
	switch cfg.Type {
	case TypeOpenAICompatible, TypeOpenAI:
		return NewOpenAICompatible(cfg, client)
	case TypeAnthropic:
		return NewAnthropic(cfg, client)
	case TypeCustomHTTP:
		return NewCustomHTTP(cfg, client)
	case TypeOpenCodeCLI:
		return NewOpenCodeCLI(cfg)
	default:
		return NewMock(cfg)
	}
}

func Enabled(configs []config.ProviderConfig) []Provider {
	var out []Provider
	for _, cfg := range configs {
		if cfg.Enabled {
			out = append(out, New(cfg))
		}
	}
	if len(out) == 0 {
		out = append(out, New(config.ProviderConfig{
			ID:           DefaultMockID,
			Type:         TypeMock,
			Enabled:      true,
			DefaultModel: DefaultMockModel,
		}))
	}
	return out
}

func Redact(configs []config.ProviderConfig) []config.ProviderConfig {
	redacted := make([]config.ProviderConfig, 0, len(configs))
	for _, cfg := range configs {
		redacted = append(redacted, cfg)
	}
	return redacted
}
