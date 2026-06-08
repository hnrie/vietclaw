package agent

import (
	"context"
	"fmt"
	"strings"

	"vietclaw/internal/config"
	"vietclaw/internal/i18n"
	"vietclaw/internal/providers"
	"vietclaw/internal/router"
)

func (s *Service) ChatStream(ctx context.Context, req ChatRequest) (<-chan providers.StreamChunk, error) {
	req = normalizeRequest(req, s.cfg)
	req = s.applyAgentProfile(ctx, req)
	if strings.TrimSpace(req.Message) == "" {
		return nil, fmt.Errorf("%s", s.text(i18n.AgentMessageRequired))
	}

	if err := s.ensureSession(ctx, req); err != nil {
		return nil, err
	}
	if err := s.addMessage(ctx, req.SessionID, RoleUser, req.Message); err != nil {
		return nil, err
	}

	intent := s.router.Classify(ctx, req.Message, s.Language())
	runID := newID("run")
	if err := s.insertRun(ctx, runID, req.SessionID, string(intent), "", "", RunStatusRunning, ""); err != nil {
		return nil, err
	}

	switch intent {
	case router.IntentMemoryAdd:
		return s.streamRuleResponse(ctx, req, runID, intent, s.handleMemoryAdd)
	case router.IntentMemoryQuery:
		return s.streamRuleResponse(ctx, req, runID, intent, s.handleMemoryQuery)
	default:
		return s.StreamAgenticLoop(ctx, req, runID, intent)
	}
}

func (s *Service) runAgenticLoop(ctx context.Context, req ChatRequest, runID string, intent router.Intent) (ChatResponse, error) {
	embedder := s.router.SelectDefaultEmbedder()
	messages, err := s.context.Messages(ctx, req.SessionID, s.memoryScope(req), req.Message, embedder)
	if err != nil {
		_ = s.finishRun(ctx, runID, RunStatusFailed, err.Error(), "", "")
		return ChatResponse{}, err
	}
	messages = s.applyProfilePersona(req, messages)

	chatReq := s.loopChatRequest(req, messages)
	selection, excludedProviders, err := s.selectLoopProvider(ctx, chatReq, nil)
	if err != nil {
		return s.selectionError(ctx, req, runID, intent, err), nil
	}
	chatReq.Model = selection.Model

	var finalReply string
	var finalProvider string
	var finalModel string
	var totalCost float64
	var accumulatedText string

	for {
		providerResp, nextSelection, nextExcluded, err := s.chatWithFallback(ctx, chatReq, selection, excludedProviders)
		if err != nil {
			_ = s.finishRun(ctx, runID, RunStatusFailed, err.Error(), selection.Provider.ID(), selection.Model)
			return ChatResponse{
				OK:        false,
				SessionID: req.SessionID,
				AgentID:   req.AgentID,
				Intent:    string(intent),
				Provider:  selection.Provider.ID(),
				Model:     selection.Model,
				Error:     err.Error(),
			}, err
		}
		selection = nextSelection
		excludedProviders = nextExcluded
		chatReq.Model = selection.Model

		totalCost += providerResp.EstimatedCostUSD
		finalProvider = providerResp.Provider
		finalModel = providerResp.Model
		if providerResp.Text != "" {
			accumulatedText += providerResp.Text
		}

		if len(providerResp.ToolCalls) > 0 {
			chatReq.Messages = append(chatReq.Messages, providers.Message{
				Role:      RoleAssistant,
				Content:   accumulatedText,
				ToolCalls: providerResp.ToolCalls,
			})
			if err := s.addMessage(ctx, req.SessionID, RoleAssistant, accumulatedText); err != nil {
				return ChatResponse{}, err
			}
			if err := s.executeToolCalls(ctx, req.SessionID, providerResp.ToolCalls, &chatReq); err != nil {
				return ChatResponse{}, err
			}
			continue
		}

		finalReply = accumulatedText
		break
	}

	_ = s.addMessage(ctx, req.SessionID, RoleAssistant, finalReply)
	_ = s.insertCost(ctx, providers.ChatResponse{
		Provider:         finalProvider,
		Model:            finalModel,
		EstimatedCostUSD: totalCost,
	})
	_ = s.finishRun(ctx, runID, RunStatusCompleted, finalReply, finalProvider, finalModel)

	return ChatResponse{
		OK:        true,
		SessionID: req.SessionID,
		AgentID:   req.AgentID,
		Intent:    string(intent),
		Reply:     finalReply,
		Provider:  finalProvider,
		Model:     finalModel,
		CostUSD:   totalCost,
	}, nil
}

func (s *Service) StreamAgenticLoop(ctx context.Context, req ChatRequest, runID string, intent router.Intent) (<-chan providers.StreamChunk, error) {
	ch := make(chan providers.StreamChunk)

	go func() {
		defer close(ch)

		embedder := s.router.SelectDefaultEmbedder()
		messages, err := s.context.Messages(ctx, req.SessionID, s.memoryScope(req), req.Message, embedder)
		if err != nil {
			_ = s.finishRun(ctx, runID, RunStatusFailed, err.Error(), "", "")
			ch <- providers.StreamChunk{Error: err.Error()}
			return
		}
		messages = s.applyProfilePersona(req, messages)

		chatReq := s.loopChatRequest(req, messages)
		selection, excludedProviders, err := s.selectLoopProvider(ctx, chatReq, nil)
		if err != nil {
			resp := s.selectionError(ctx, req, runID, intent, err)
			ch <- providers.StreamChunk{Error: resp.Error}
			return
		}
		chatReq.Model = selection.Model

		var finalProvider string
		var finalModel string
		var totalCost float64
		var accumulatedText string

		for {
			attempt, nextSelection, nextExcluded, err := s.streamWithFallback(ctx, ch, chatReq, selection, excludedProviders)
			if err != nil {
				_ = s.finishRun(ctx, runID, RunStatusFailed, err.Error(), selection.Provider.ID(), selection.Model)
				ch <- providers.StreamChunk{Error: err.Error()}
				return
			}
			selection = nextSelection
			excludedProviders = nextExcluded
			chatReq.Model = selection.Model

			finalProvider = selection.Provider.ID()
			finalModel = selection.Model
			if attempt.text != "" {
				accumulatedText += attempt.text
			}

			tempReq := chatReq
			tempReq.MaxOutputTokens = providers.EstimateTokens(accumulatedText)
			totalCost += selection.Provider.EstimateCost(tempReq).EstimatedCostUSD

			if len(attempt.toolCalls) > 0 {
				chatReq.Messages = append(chatReq.Messages, providers.Message{
					Role:      RoleAssistant,
					Content:   accumulatedText,
					ToolCalls: attempt.toolCalls,
				})
				if err := s.addMessage(ctx, req.SessionID, RoleAssistant, accumulatedText); err != nil {
					ch <- providers.StreamChunk{Error: err.Error()}
					return
				}

				for _, tc := range attempt.toolCalls {
					ch <- providers.StreamChunk{
						Event:     "tool_call",
						ToolName:  tc.Function.Name,
						ToolInput: tc.Function.Arguments,
					}
					toolResult, err := s.executeToolCall(ctx, req.SessionID, tc, &chatReq)
					if err != nil {
						ch <- providers.StreamChunk{Error: err.Error()}
						return
					}
					ch <- providers.StreamChunk{
						Event:      "tool_result",
						ToolName:   tc.Function.Name,
						ToolResult: toolResult,
					}
				}
				continue
			}
			break
		}

		_ = s.addMessage(ctx, req.SessionID, RoleAssistant, accumulatedText)
		_ = s.insertCost(ctx, providers.ChatResponse{
			Provider:         finalProvider,
			Model:            finalModel,
			EstimatedCostUSD: totalCost,
		})
		_ = s.finishRun(ctx, runID, RunStatusCompleted, accumulatedText, finalProvider, finalModel)
		ch <- providers.StreamChunk{Done: true}
	}()

	return ch, nil
}

type ruleHandler func(context.Context, ChatRequest, string, router.Intent) (ChatResponse, error)

func (s *Service) streamRuleResponse(ctx context.Context, req ChatRequest, runID string, intent router.Intent, handle ruleHandler) (<-chan providers.StreamChunk, error) {
	ch := make(chan providers.StreamChunk, 2)
	go func() {
		defer close(ch)
		resp, err := handle(ctx, req, runID, intent)
		if err != nil {
			ch <- providers.StreamChunk{Error: err.Error()}
			return
		}
		ch <- providers.StreamChunk{Text: resp.Reply}
		ch <- providers.StreamChunk{Done: true}
	}()
	return ch, nil
}

func (s *Service) loopChatRequest(req ChatRequest, messages []providers.Message) providers.ChatRequest {
	return providers.ChatRequest{
		SessionID:       req.SessionID,
		Messages:        messages,
		Temperature:     defaultTemperature,
		MaxOutputTokens: s.maxOutputTokens(),
		Metadata: map[string]any{
			"user_id":  req.UserID,
			"channel":  req.Channel,
			"mode":     req.Mode,
			"language": s.Language(),
		},
		Tools: s.tools.GetDefinitions(),
	}
}

func (s *Service) maxOutputTokens() int {
	if s.cfg.Agent.MaxOutputTokens > 0 {
		return s.cfg.Agent.MaxOutputTokens
	}
	return config.DefaultMaxOutputTokens
}

func (s *Service) selectLoopProvider(ctx context.Context, req providers.ChatRequest, excluded []string) (router.Selection, []string, error) {
	selection, err := s.router.Select(ctx, req, excluded)
	return selection, excluded, err
}

func (s *Service) selectionError(ctx context.Context, req ChatRequest, runID string, intent router.Intent, err error) ChatResponse {
	reply := err.Error()
	_ = s.addMessage(ctx, req.SessionID, RoleAssistant, reply)
	_ = s.finishRun(ctx, runID, RunStatusNeedsApproval, reply, "", "")
	return ChatResponse{
		OK:        false,
		SessionID: req.SessionID,
		AgentID:   req.AgentID,
		Intent:    string(intent),
		Reply:     reply,
		Error:     reply,
	}
}

func (s *Service) chatWithFallback(ctx context.Context, req providers.ChatRequest, selection router.Selection, excluded []string) (providers.ChatResponse, router.Selection, []string, error) {
	for {
		resp, err := selection.Provider.Chat(ctx, req)
		if err == nil {
			return resp, selection, excluded, nil
		}
		s.logf("provider %s chat error: %v", selection.Provider.ID(), err)
		excluded = append(excluded, selection.Provider.ID())
		next, fallbackErr := s.router.Select(ctx, req, excluded)
		if fallbackErr != nil {
			return providers.ChatResponse{}, selection, excluded, err
		}
		selection = next
		req.Model = selection.Model
	}
}

type streamAttempt struct {
	text      string
	toolCalls []providers.ToolCall
}

func (s *Service) streamWithFallback(ctx context.Context, out chan<- providers.StreamChunk, req providers.ChatRequest, selection router.Selection, excluded []string) (streamAttempt, router.Selection, []string, error) {
	for {
		streamCh, err := selection.Provider.ChatStream(ctx, req)
		if err != nil {
			s.logf("provider %s stream init error: %v", selection.Provider.ID(), err)
			excluded = append(excluded, selection.Provider.ID())
			next, fallbackErr := s.router.Select(ctx, req, excluded)
			if fallbackErr != nil {
				return streamAttempt{}, selection, excluded, err
			}
			selection = next
			req.Model = selection.Model
			continue
		}

		var attempt streamAttempt
		failed := ""
		for chunk := range streamCh {
			if chunk.Error != "" {
				failed = chunk.Error
				break
			}
			if chunk.Text != "" {
				attempt.text += chunk.Text
				out <- providers.StreamChunk{Event: "text", Text: chunk.Text}
			}
			if len(chunk.ToolCalls) > 0 {
				attempt.toolCalls = append(attempt.toolCalls, chunk.ToolCalls...)
			}
		}
		if failed == "" {
			return attempt, selection, excluded, nil
		}

		s.logf("provider %s stream generation error: %s", selection.Provider.ID(), failed)
		excluded = append(excluded, selection.Provider.ID())
		next, fallbackErr := s.router.Select(ctx, req, excluded)
		if fallbackErr != nil {
			return streamAttempt{}, selection, excluded, fmt.Errorf("%s", failed)
		}
		selection = next
		req.Model = selection.Model
	}
}

func (s *Service) executeToolCalls(ctx context.Context, sessionID string, calls []providers.ToolCall, req *providers.ChatRequest) error {
	for _, tc := range calls {
		if _, err := s.executeToolCall(ctx, sessionID, tc, req); err != nil {
			return err
		}
	}
	return nil
}

func (s *Service) executeToolCall(ctx context.Context, sessionID string, tc providers.ToolCall, req *providers.ChatRequest) (string, error) {
	toolResult, err := s.tools.Execute(ctx, tc.Function.Name, tc.Function.Arguments)
	if err != nil {
		toolResult = s.text(i18n.AgentToolExecuteError, err.Error())
	}

	req.Messages = append(req.Messages, providers.Message{
		Role:       "tool",
		Name:       tc.Function.Name,
		ToolCallID: tc.ID,
		Content:    toolResult,
	})

	toolPersistText := fmt.Sprintf("[Tool Execution: %s]\nInput: %s\nOutput: %s", tc.Function.Name, tc.Function.Arguments, toolResult)
	if err := s.addMessage(ctx, sessionID, "system", toolPersistText); err != nil {
		return "", err
	}
	return toolResult, nil
}
