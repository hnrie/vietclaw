package memory_test

import (
	"context"

	"vietclaw/internal/providers"
)

type fixedEmbedder struct {
	embedding []float32
}

func (f fixedEmbedder) ID() string   { return "fixed" }
func (f fixedEmbedder) Type() string { return providers.TypeMock }

func (f fixedEmbedder) Chat(context.Context, providers.ChatRequest) (providers.ChatResponse, error) {
	return providers.ChatResponse{}, nil
}

func (f fixedEmbedder) ChatStream(context.Context, providers.ChatRequest) (<-chan providers.StreamChunk, error) {
	ch := make(chan providers.StreamChunk)
	close(ch)
	return ch, nil
}

func (f fixedEmbedder) Embed(context.Context, string) ([]float32, error) {
	return f.embedding, nil
}

func (f fixedEmbedder) EstimateCost(providers.ChatRequest) providers.CostEstimate {
	return providers.CostEstimate{}
}
