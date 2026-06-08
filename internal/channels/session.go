package channels

const (
	PlatformDiscord  = "discord"
	PlatformTelegram = "telegram"
)

func SessionKey(msg InboundMessage) string {
	switch msg.Platform {
	case PlatformDiscord:
		if msg.IsDM {
			return "discord:dm:" + msg.UserID
		}
		if msg.ThreadID != "" {
			return "discord:guild:" + msg.GuildID + ":thread:" + msg.ThreadID
		}
		return "discord:guild:" + msg.GuildID + ":channel:" + msg.ChannelID
	case PlatformTelegram:
		if msg.IsDM {
			return "telegram:private:" + msg.UserID
		}
		if msg.ThreadID != "" {
			return "telegram:chat:" + msg.ChatID + ":topic:" + msg.ThreadID
		}
		return "telegram:chat:" + msg.ChatID
	default:
		if msg.IsDM {
			return msg.Platform + ":dm:" + msg.UserID
		}
		return msg.Platform + ":chat:" + defaultString(msg.ChatID, msg.ChannelID)
	}
}

func UserIdentity(msg InboundMessage) string {
	if msg.IsGroup {
		groupID := defaultString(msg.GuildID, msg.ChatID)
		return msg.Platform + ":group:" + groupID + ":user:" + msg.UserID
	}
	return msg.Platform + ":" + msg.UserID
}
