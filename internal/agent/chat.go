package agent

import (
	"context"
	"fmt"
	"strings"

	"vietclaw/internal/i18n"
	"vietclaw/internal/router"
)

func (s *Service) Chat(ctx context.Context, req ChatRequest) (ChatResponse, error) {
	req = normalizeRequest(req, s.cfg)
	if strings.TrimSpace(req.Message) == "" {
		errText := s.text(i18n.AgentMessageRequired)
		return ChatResponse{
			OK:        false,
			SessionID: req.SessionID,
			Intent:    string(router.IntentUnknown),
			Error:     errText,
		}, fmt.Errorf("%s", errText)
	}

	if err := s.ensureSession(ctx, req); err != nil {
		return ChatResponse{}, err
	}
	if err := s.addMessage(ctx, req.SessionID, RoleUser, req.Message); err != nil {
		return ChatResponse{}, err
	}

	intent := router.Classify(req.Message)
	runID := newID("run")
	if err := s.insertRun(ctx, runID, req.SessionID, string(intent), "", "", RunStatusRunning, ""); err != nil {
		return ChatResponse{}, err
	}

	switch intent {
	case router.IntentMemoryAdd:
		return s.handleMemoryAdd(ctx, req, runID, intent)
	case router.IntentMemoryQuery:
		return s.handleMemoryQuery(ctx, req, runID, intent)
	case router.IntentAction:
		return s.runAgenticLoop(ctx, req, runID, intent)
	default:
		return s.runAgenticLoop(ctx, req, runID, intent)
	}
}
