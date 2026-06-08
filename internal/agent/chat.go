package agent

import (
	"context"
	"fmt"
	"strings"

	"vietclaw/internal/router"
)

const actionBlockedReply = "tool action cần policy rõ hơn. shell.exec đang tắt mặc định nếu chưa bật trong config."

func (s *Service) Chat(ctx context.Context, req ChatRequest) (ChatResponse, error) {
	req = normalizeRequest(req, s.cfg)
	if strings.TrimSpace(req.Message) == "" {
		return ChatResponse{
			OK:        false,
			SessionID: req.SessionID,
			Intent:    string(router.IntentUnknown),
			Error:     "message is required",
		}, fmt.Errorf("message is required")
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
		_ = s.addMessage(ctx, req.SessionID, RoleAssistant, actionBlockedReply)
		_ = s.finishRun(ctx, runID, RunStatusBlocked, "tool policy blocked", ProviderLocal, ModelRule)
		return ChatResponse{
			OK:        true,
			SessionID: req.SessionID,
			Intent:    string(intent),
			Reply:     actionBlockedReply,
			Provider:  ProviderLocal,
			Model:     ModelRule,
		}, nil
	default:
		return s.handleProviderChat(ctx, req, runID, intent)
	}
}
