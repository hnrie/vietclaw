package discord

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/bwmarrin/discordgo"

	"vietclaw/internal/channels"
	"vietclaw/internal/config"
)

type Adapter struct {
	cfg     config.DiscordConfig
	handler *channels.Handler
}

func New(cfg config.DiscordConfig, handler *channels.Handler) *Adapter {
	return &Adapter{cfg: cfg, handler: handler}
}

func (a *Adapter) Name() string {
	return "discord"
}

func (a *Adapter) Start(ctx context.Context) error {
	token := strings.TrimSpace(os.Getenv(a.cfg.TokenEnv))
	if token == "" {
		return fmt.Errorf("discord token env missing: %s", a.cfg.TokenEnv)
	}
	session, err := discordgo.New("Bot " + token)
	if err != nil {
		return fmt.Errorf("create discord session: %w", err)
	}
	session.Identify.Intents = discordgo.IntentsGuildMessages | discordgo.IntentsDirectMessages | discordgo.IntentMessageContent

	session.AddHandler(a.onMessage(ctx))
	if err := session.Open(); err != nil {
		return fmt.Errorf("open discord session: %w", err)
	}
	defer session.Close()

	<-ctx.Done()
	return nil
}

func (a *Adapter) onMessage(ctx context.Context) func(*discordgo.Session, *discordgo.MessageCreate) {
	return func(s *discordgo.Session, event *discordgo.MessageCreate) {
		msg := event.Message
		if msg == nil || msg.Author == nil || msg.Author.Bot || strings.TrimSpace(msg.Content) == "" {
			return
		}
		if s.State != nil && s.State.User != nil && msg.Author.ID == s.State.User.ID {
			return
		}
		if msg.GuildID != "" && !channels.Allowed(msg.GuildID, a.cfg.AllowedGuilds) {
			return
		}
		if !channels.Allowed(msg.ChannelID, a.cfg.AllowedChannels) {
			return
		}

		botID := ""
		if s.State != nil && s.State.User != nil {
			botID = s.State.User.ID
		}
		mentionsBot := mentionsUser(msg, botID)
		isReplyToBot := msg.ReferencedMessage != nil && msg.ReferencedMessage.Author != nil && msg.ReferencedMessage.Author.ID == botID
		isDM := msg.GuildID == ""
		threadID := ""
		if msg.Thread != nil {
			threadID = msg.Thread.ID
		}

		inbound := channels.InboundMessage{
			Platform:     "discord",
			MessageID:    msg.ID,
			GuildID:      msg.GuildID,
			ChannelID:    msg.ChannelID,
			ThreadID:     threadID,
			UserID:       msg.Author.ID,
			Username:     msg.Author.Username,
			IsDM:         isDM,
			IsGroup:      !isDM,
			IsReplyToBot: isReplyToBot,
			MentionsBot:  mentionsBot,
			Text:         msg.Content,
			RawText:      msg.Content,
			CreatedAt:    time.Now().UTC(),
		}

		if !channels.ShouldHandle(inbound, channels.DiscordPolicy(a.cfg)) {
			return
		}
		_ = s.ChannelTyping(msg.ChannelID)
		mentions := []string{"<@" + botID + ">", "<@!" + botID + ">"}
		_ = a.handler.Handle(ctx, inbound, channels.DiscordPolicy(a.cfg), mentions, func(sendCtx context.Context, replyTo channels.InboundMessage, reply string) error {
			return sendChunks(sendCtx, s, replyTo.ChannelID, replyTo.MessageID, reply)
		})
	}
}

func mentionsUser(msg *discordgo.Message, userID string) bool {
	if userID == "" {
		return false
	}
	for _, mention := range msg.Mentions {
		if mention != nil && mention.ID == userID {
			return true
		}
	}
	return strings.Contains(msg.Content, "<@"+userID+">") || strings.Contains(msg.Content, "<@!"+userID+">")
}

func sendChunks(ctx context.Context, session *discordgo.Session, channelID, referenceID, text string) error {
	for _, chunk := range chunks(text, 1900) {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}
		message := &discordgo.MessageSend{Content: chunk}
		if referenceID != "" {
			message.Reference = &discordgo.MessageReference{MessageID: referenceID, ChannelID: channelID}
		}
		if _, err := session.ChannelMessageSendComplex(channelID, message); err != nil {
			return err
		}
	}
	return nil
}

func chunks(text string, limit int) []string {
	runes := []rune(strings.TrimSpace(text))
	if len(runes) == 0 {
		return []string{"gọi t rồi muốn t làm gì?"}
	}
	var out []string
	for len(runes) > 0 {
		n := limit
		if len(runes) < n {
			n = len(runes)
		}
		out = append(out, string(runes[:n]))
		runes = runes[n:]
	}
	return out
}
