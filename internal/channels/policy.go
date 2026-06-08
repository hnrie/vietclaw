package channels

import (
	"strings"

	"vietclaw/internal/config"
)

const (
	EmptyPromptReply = "gọi t rồi muốn t làm gì?"
	EmptyAgentReply  = "t chưa có gì để trả lời."
)

func ShouldHandle(msg InboundMessage, policy Policy) bool {
	if msg.IsDM {
		return policy.RespondInDM
	}
	return msg.MentionsBot || msg.IsReplyToBot
}

func StripMentions(text string, mentions []string) string {
	cleaned := strings.TrimSpace(text)
	for {
		changed := false
		for _, mention := range mentions {
			if mention == "" {
				continue
			}
			if strings.HasPrefix(strings.ToLower(cleaned), strings.ToLower(mention)) {
				cleaned = strings.TrimSpace(cleaned[len(mention):])
				changed = true
			}
		}
		if !changed {
			return cleaned
		}
	}
}

func DiscordPolicy(cfg config.DiscordConfig) Policy {
	return Policy{RespondInDM: cfg.RespondInDM, RespondInGroups: cfg.RespondInGuilds}
}

func TelegramPolicy(cfg config.TelegramConfig) Policy {
	return Policy{RespondInDM: cfg.RespondInPrivate, RespondInGroups: cfg.RespondInGroups}
}

func Allowed(value string, allowed []string) bool {
	if len(allowed) == 0 {
		return true
	}
	for _, item := range allowed {
		if value == item {
			return true
		}
	}
	return false
}
