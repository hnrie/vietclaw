package agent

import (
	"context"
	"fmt"
	"strings"

	"vietclaw/internal/providers"
	"vietclaw/internal/router"
)

const maxAgenticSteps = 5

func (s *Service) ChatStream(ctx context.Context, req ChatRequest) (<-chan providers.StreamChunk, error) {
	req = normalizeRequest(req, s.cfg)
	if strings.TrimSpace(req.Message) == "" {
		return nil, fmt.Errorf("message is empty")
	}

	if err := s.ensureSession(ctx, req); err != nil {
		return nil, err
	}
	if err := s.addMessage(ctx, req.SessionID, RoleUser, req.Message); err != nil {
		return nil, err
	}

	intent := router.Classify(req.Message)
	runID := newID("run")
	if err := s.insertRun(ctx, runID, req.SessionID, string(intent), "", "", RunStatusRunning, ""); err != nil {
		return nil, err
	}

	switch intent {
	case router.IntentMemoryAdd:
		ch := make(chan providers.StreamChunk, 2)
		go func() {
			defer close(ch)
			resp, err := s.handleMemoryAdd(ctx, req, runID, intent)
			if err != nil {
				ch <- providers.StreamChunk{Error: err.Error()}
				return
			}
			ch <- providers.StreamChunk{Text: resp.Reply}
			ch <- providers.StreamChunk{Done: true}
		}()
		return ch, nil

	case router.IntentMemoryQuery:
		ch := make(chan providers.StreamChunk, 2)
		go func() {
			defer close(ch)
			resp, err := s.handleMemoryQuery(ctx, req, runID, intent)
			if err != nil {
				ch <- providers.StreamChunk{Error: err.Error()}
				return
			}
			ch <- providers.StreamChunk{Text: resp.Reply}
			ch <- providers.StreamChunk{Done: true}
		}()
		return ch, nil

	default:
		return s.StreamAgenticLoop(ctx, req, runID, intent)
	}
}


func (s *Service) runAgenticLoop(ctx context.Context, req ChatRequest, runID string, intent router.Intent) (ChatResponse, error) {
	embedder := s.router.SelectDefaultEmbedder()
	messages, err := s.context.Messages(ctx, req.SessionID, req.UserID, req.Message, embedder)
	if err != nil {
		_ = s.finishRun(ctx, runID, RunStatusFailed, err.Error(), "", "")
		return ChatResponse{}, err
	}

	chatReq := providers.ChatRequest{
		SessionID:       req.SessionID,
		Messages:        messages,
		Temperature:     defaultTemperature,
		MaxOutputTokens: defaultMaxOutputTokens,
		Metadata: map[string]any{
			"user_id":  req.UserID,
			"channel":  req.Channel,
			"mode":     req.Mode,
			"language": s.Language(),
		},
		Tools: s.tools.GetDefinitions(),
	}

	var excludedProviders []string
	var selection router.Selection

	for {
		sel, err := s.router.Select(ctx, chatReq, excludedProviders)
		if err != nil {
			reply := err.Error()
			_ = s.addMessage(ctx, req.SessionID, RoleAssistant, reply)
			_ = s.finishRun(ctx, runID, RunStatusNeedsApproval, reply, "", "")
			return ChatResponse{
				OK:        false,
				SessionID: req.SessionID,
				Intent:    string(intent),
				Reply:     reply,
				Error:     reply,
			}, nil
		}
		selection = sel
		break
	}
	chatReq.Model = selection.Model

	var finalReply string
	var finalProvider string
	var finalModel string
	var totalCost float64

	for step := 1; step <= maxAgenticSteps; step++ {
		var providerResp providers.ChatResponse
		var err error

		for {
			providerResp, err = selection.Provider.Chat(ctx, chatReq)
			if err != nil {
				// Provider failed, exclude it and fallback
				excludedProviders = append(excludedProviders, selection.Provider.ID())
				sel, fallbackErr := s.router.Select(ctx, chatReq, excludedProviders)
				if fallbackErr != nil {
					_ = s.finishRun(ctx, runID, RunStatusFailed, err.Error(), selection.Provider.ID(), selection.Model)
					return ChatResponse{
						OK:        false,
						SessionID: req.SessionID,
						Intent:    string(intent),
						Provider:  selection.Provider.ID(),
						Model:     selection.Model,
						Error:     err.Error(),
					}, err
				}
				selection = sel
				chatReq.Model = selection.Model
				continue
			}
			break
		}

		totalCost += providerResp.EstimatedCostUSD
		finalProvider = providerResp.Provider
		finalModel = providerResp.Model

		if len(providerResp.ToolCalls) > 0 {
			assistantMsg := providers.Message{
				Role:      RoleAssistant,
				Content:   providerResp.Text,
				ToolCalls: providerResp.ToolCalls,
			}
			chatReq.Messages = append(chatReq.Messages, assistantMsg)
			if err := s.addMessage(ctx, req.SessionID, RoleAssistant, providerResp.Text); err != nil {
				return ChatResponse{}, err
			}

			for _, tc := range providerResp.ToolCalls {
				toolResult, err := s.tools.Execute(ctx, tc.Function.Name, tc.Function.Arguments)
				if err != nil {
					toolResult = fmt.Sprintf("Error executing tool: %s", err.Error())
				}

				toolMsg := providers.Message{
					Role:       "tool",
					Name:       tc.Function.Name,
					ToolCallID: tc.ID,
					Content:    toolResult,
				}
				chatReq.Messages = append(chatReq.Messages, toolMsg)

				toolPersistText := fmt.Sprintf("[Tool Execution: %s]\nInput: %s\nOutput: %s", tc.Function.Name, tc.Function.Arguments, toolResult)
				if err := s.addMessage(ctx, req.SessionID, "system", toolPersistText); err != nil {
					return ChatResponse{}, err
				}
			}

			continue
		}

		finalReply = providerResp.Text
		break
	}

	if finalReply == "" {
		finalReply = "Đã đạt số bước thực thi tối đa nhưng chưa có phản hồi cuối cùng."
	}

	_ = s.addMessage(ctx, req.SessionID, RoleAssistant, finalReply)

	accumulatedResp := providers.ChatResponse{
		Provider:         finalProvider,
		Model:            finalModel,
		EstimatedCostUSD: totalCost,
	}
	_ = s.insertCost(ctx, accumulatedResp)
	_ = s.finishRun(ctx, runID, RunStatusCompleted, finalReply, finalProvider, finalModel)

	return ChatResponse{
		OK:        true,
		SessionID: req.SessionID,
		Intent:    string(intent),
		Reply:     finalReply,
		Provider:  finalProvider,
		Model:     finalModel,
		CostUSD:   totalCost,
	}, nil
}

func (s *Service) StreamAgenticLoop(ctx context.Context, req ChatRequest, runID string, intent router.Intent) (<-chan providers.StreamChunk, error) {
	ch := make(chan providers.StreamChunk, 64)

	go func() {
		defer close(ch)

		embedder := s.router.SelectDefaultEmbedder()
		messages, err := s.context.Messages(ctx, req.SessionID, req.UserID, req.Message, embedder)
		if err != nil {
			_ = s.finishRun(ctx, runID, RunStatusFailed, err.Error(), "", "")
			ch <- providers.StreamChunk{Error: err.Error()}
			return
		}

		chatReq := providers.ChatRequest{
			SessionID:       req.SessionID,
			Messages:        messages,
			Temperature:     defaultTemperature,
			MaxOutputTokens: defaultMaxOutputTokens,
			Metadata: map[string]any{
				"user_id":  req.UserID,
				"channel":  req.Channel,
				"mode":     req.Mode,
				"language": s.Language(),
			},
			Tools: s.tools.GetDefinitions(),
		}

		var excludedProviders []string
		var selection router.Selection

		for {
			sel, err := s.router.Select(ctx, chatReq, excludedProviders)
			if err != nil {
				reply := err.Error()
				_ = s.addMessage(ctx, req.SessionID, RoleAssistant, reply)
				_ = s.finishRun(ctx, runID, RunStatusNeedsApproval, reply, "", "")
				ch <- providers.StreamChunk{Error: reply}
				return
			}
			selection = sel
			break
		}
		chatReq.Model = selection.Model

		var finalProvider string
		var finalModel string
		var totalCost float64
		var currentAssistantMessage string

		for step := 1; step <= maxAgenticSteps; step++ {
			var streamCh <-chan providers.StreamChunk
			var err error

			for {
				streamCh, err = selection.Provider.ChatStream(ctx, chatReq)
				if err != nil {
					excludedProviders = append(excludedProviders, selection.Provider.ID())
					sel, fallbackErr := s.router.Select(ctx, chatReq, excludedProviders)
					if fallbackErr != nil {
						_ = s.finishRun(ctx, runID, RunStatusFailed, err.Error(), selection.Provider.ID(), selection.Model)
						ch <- providers.StreamChunk{Error: err.Error()}
						return
					}
					selection = sel
					chatReq.Model = selection.Model
					continue
				}
				break
			}

			finalProvider = selection.Provider.ID()
			finalModel = selection.Model
			currentAssistantMessage = ""

			var stepToolCalls []providers.ToolCall

			for chunk := range streamCh {
				if chunk.Error != "" {
					ch <- providers.StreamChunk{Error: chunk.Error}
					return
				}
				if chunk.Text != "" {
					currentAssistantMessage += chunk.Text
					ch <- providers.StreamChunk{Text: chunk.Text}
				}
				if len(chunk.ToolCalls) > 0 {
					stepToolCalls = append(stepToolCalls, chunk.ToolCalls...)
				}
			}

			// Estimate token cost
			outTokens := providers.EstimateTokens(currentAssistantMessage)
			tempReq := chatReq
			tempReq.MaxOutputTokens = outTokens
			totalCost += selection.Provider.EstimateCost(tempReq).EstimatedCostUSD

			if len(stepToolCalls) > 0 {
				assistantMsg := providers.Message{
					Role:      RoleAssistant,
					Content:   currentAssistantMessage,
					ToolCalls: stepToolCalls,
				}
				chatReq.Messages = append(chatReq.Messages, assistantMsg)
				if err := s.addMessage(ctx, req.SessionID, RoleAssistant, currentAssistantMessage); err != nil {
					ch <- providers.StreamChunk{Error: err.Error()}
					return
				}

				// Execute tool calls
				for _, tc := range stepToolCalls {
					statusMsg := fmt.Sprintf("\n*[Chạy công cụ: %s...]*\n", tc.Function.Name)
					ch <- providers.StreamChunk{Text: statusMsg}

					toolResult, err := s.tools.Execute(ctx, tc.Function.Name, tc.Function.Arguments)
					if err != nil {
						toolResult = fmt.Sprintf("Lỗi thực thi công cụ: %s", err.Error())
					}

					toolMsg := providers.Message{
						Role:       "tool",
						Name:       tc.Function.Name,
						ToolCallID: tc.ID,
						Content:    toolResult,
					}
					chatReq.Messages = append(chatReq.Messages, toolMsg)

					toolPersistText := fmt.Sprintf("[Tool Execution: %s]\nInput: %s\nOutput: %s", tc.Function.Name, tc.Function.Arguments, toolResult)
					if err := s.addMessage(ctx, req.SessionID, "system", toolPersistText); err != nil {
						ch <- providers.StreamChunk{Error: err.Error()}
						return
					}

					ch <- providers.StreamChunk{Text: fmt.Sprintf("\n*[Kết quả: %s]*\n", toolResult)}
				}

				continue
			}

			break
		}

		if currentAssistantMessage == "" {
			currentAssistantMessage = "Đã đạt số bước thực thi tối đa nhưng chưa có phản hồi cuối cùng."
			ch <- providers.StreamChunk{Text: currentAssistantMessage}
		}

		_ = s.addMessage(ctx, req.SessionID, RoleAssistant, currentAssistantMessage)

		accumulatedResp := providers.ChatResponse{
			Provider:         finalProvider,
			Model:            finalModel,
			EstimatedCostUSD: totalCost,
		}
		_ = s.insertCost(ctx, accumulatedResp)
		_ = s.finishRun(ctx, runID, RunStatusCompleted, currentAssistantMessage, finalProvider, finalModel)

		ch <- providers.StreamChunk{Done: true}
	}()

	return ch, nil
}
