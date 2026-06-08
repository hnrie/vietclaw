package agent

import (
	"context"
	"strings"

	"vietclaw/internal/config"
	"vietclaw/internal/providers"
)

func (s *Service) profile(agentID string) config.AgentProfileConfig {
	if agentID == "" {
		agentID = config.DefaultAgentID
	}
	for _, profile := range s.cfg.Agents {
		if profile.ID == agentID {
			return profile
		}
	}
	return config.AgentProfileConfig{
		ID:          config.DefaultAgentID,
		Name:        s.cfg.Agent.Name,
		Language:    s.cfg.Agent.Language,
		MemoryScope: "",
	}
}

func (s *Service) memoryScope(req ChatRequest) string {
	profile := s.profile(req.AgentID)
	if profile.MemoryScope != "" {
		return profile.MemoryScope + ":user:" + req.UserID
	}
	if profile.ID != "" && profile.ID != config.DefaultAgentID {
		return "agent:" + profile.ID + ":user:" + req.UserID
	}
	return "user:" + req.UserID
}

func (s *Service) selectDelegateAgent(message string) string {
	return s.router.SelectAgent(context.Background(), message, s.Language(), s.cfg.Agents)
}

func (s *Service) applyAgentProfile(ctx context.Context, req ChatRequest) ChatRequest {
	if delegate := s.router.SelectAgent(ctx, req.Message, s.Language(), s.cfg.Agents); delegate != "" {
		req.AgentID = delegate
	}
	return req
}

func (s *Service) applyProfilePersona(req ChatRequest, messages []providers.Message) []providers.Message {
	profile := s.profile(req.AgentID)
	if strings.TrimSpace(profile.Persona) == "" || len(messages) == 0 || messages[0].Role != "system" {
		return messages
	}
	messages[0].Content += "\n\nAgent persona:\n" + profile.Persona
	return messages
}
