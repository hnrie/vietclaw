package providers

import "vietclaw/internal/config"

const (
	charsPerEstimatedToken = 4
	defaultMaxOutputTokens = 512
)

func EstimateMessagesTokens(messages []Message) int {
	total := 0
	for _, msg := range messages {
		total += EstimateTokens(msg.Content)
	}
	return total
}

func EstimateTokens(text string) int {
	n := len([]rune(text)) / charsPerEstimatedToken
	if n < 1 && text != "" {
		return 1
	}
	return n
}

func EstimateCostUSD(inTokens, outTokens int, cfg config.ProviderConfig) float64 {
	return (float64(inTokens)/1000)*cfg.CostPer1KIn + (float64(outTokens)/1000)*cfg.CostPer1KOut
}

func defaultOutputTokens(value int) int {
	if value > 0 {
		return value
	}
	return defaultMaxOutputTokens
}
