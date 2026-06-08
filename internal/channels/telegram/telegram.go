package telegram

import (
	"context"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"

	"vietclaw/internal/channels"
	"vietclaw/internal/config"
)

type Adapter struct {
	cfg     config.TelegramConfig
	handler *channels.Handler
}

func New(cfg config.TelegramConfig, handler *channels.Handler) *Adapter {
	return &Adapter{cfg: cfg, handler: handler}
}

func (a *Adapter) Name() string {
	return "telegram"
}

func (a *Adapter) Start(ctx context.Context) error {
	token := strings.TrimSpace(os.Getenv(a.cfg.TokenEnv))
	if token == "" {
		return fmt.Errorf("telegram token env missing: %s", a.cfg.TokenEnv)
	}
	bot, err := tgbotapi.NewBotAPI(token)
	if err != nil {
		return fmt.Errorf("create telegram bot: %w", err)
	}
	updateConfig := tgbotapi.NewUpdate(0)
	if a.cfg.PollTimeoutSeconds > 0 {
		updateConfig.Timeout = a.cfg.PollTimeoutSeconds
	}
	updates := bot.GetUpdatesChan(updateConfig)
	for {
		select {
		case <-ctx.Done():
			bot.StopReceivingUpdates()
			return nil
		case update := <-updates:
			if update.Message != nil {
				a.handleMessage(ctx, bot, update.Message)
			}
		}
	}
}

func (a *Adapter) handleMessage(ctx context.Context, bot *tgbotapi.BotAPI, msg *tgbotapi.Message) {
	if msg.From == nil || msg.From.IsBot || strings.TrimSpace(msg.Text) == "" {
		return
	}
	chatID := strconv.FormatInt(msg.Chat.ID, 10)
	if !channels.Allowed(chatID, a.cfg.AllowedChats) {
		return
	}
	isPrivate := msg.Chat.IsPrivate()
	botUsername := "@" + bot.Self.UserName
	mentionsBot := strings.Contains(strings.ToLower(msg.Text), strings.ToLower(botUsername))
	isReplyToBot := msg.ReplyToMessage != nil && msg.ReplyToMessage.From != nil && msg.ReplyToMessage.From.ID == bot.Self.ID

	inbound := channels.InboundMessage{
		Platform:     "telegram",
		MessageID:    strconv.Itoa(msg.MessageID),
		ChatID:       chatID,
		UserID:       strconv.FormatInt(msg.From.ID, 10),
		Username:     msg.From.UserName,
		IsDM:         isPrivate,
		IsGroup:      !isPrivate,
		IsReplyToBot: isReplyToBot,
		MentionsBot:  mentionsBot,
		Text:         msg.Text,
		RawText:      msg.Text,
		CreatedAt:    time.Now().UTC(),
	}

	if !channels.ShouldHandle(inbound, channels.TelegramPolicy(a.cfg)) {
		return
	}
	typing := tgbotapi.NewChatAction(msg.Chat.ID, tgbotapi.ChatTyping)
	_, _ = bot.Send(typing)

	_ = a.handler.Handle(ctx, inbound, channels.TelegramPolicy(a.cfg), []string{botUsername}, func(sendCtx context.Context, replyTo channels.InboundMessage, reply string) error {
		return sendChunks(sendCtx, bot, msg, reply)
	})
}

func sendChunks(ctx context.Context, bot *tgbotapi.BotAPI, replyTo *tgbotapi.Message, text string) error {
	for _, chunk := range chunks(text, 3900) {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}
		msg := tgbotapi.NewMessage(replyTo.Chat.ID, chunk)
		if !replyTo.Chat.IsPrivate() {
			msg.ReplyToMessageID = replyTo.MessageID
		}
		if _, err := bot.Send(msg); err != nil {
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
