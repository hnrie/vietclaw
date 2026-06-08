package router

import "vietclaw/internal/providers"

func (r *ModelRouter) defaultProvider(excludeIDs []string) providers.Provider {
	isExcluded := func(id string) bool {
		for _, ex := range excludeIDs {
			if ex == id {
				return true
			}
		}
		return false
	}

	for _, p := range r.providers {
		if p.ID() == r.cfg.Router.DefaultProvider && !isExcluded(p.ID()) {
			return p
		}
	}
	if r.cfg.Router.CheapFirst {
		for _, p := range r.providers {
			if p.Type() == providers.TypeMock && !isExcluded(p.ID()) {
				return p
			}
		}
	}
	for _, p := range r.providers {
		if !isExcluded(p.ID()) {
			return p
		}
	}
	return nil
}

func (r *ModelRouter) defaultModel(provider providers.Provider) string {
	for _, cfg := range r.cfg.Providers {
		if cfg.ID == provider.ID() && cfg.DefaultModel != "" {
			return cfg.DefaultModel
		}
	}
	if r.cfg.Router.DefaultModel != "" {
		return r.cfg.Router.DefaultModel
	}
	return providers.DefaultMockModel
}
