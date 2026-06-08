package channels

import (
	"context"
	"time"
)

type Adapter interface {
	Name() string
	Start(ctx context.Context) error
}

type Sender func(ctx context.Context, msg InboundMessage, reply string) error

type InboundMessage struct {
	Platform     string
	MessageID    string
	GuildID      string
	ChannelID    string
	ThreadID     string
	ChatID       string
	UserID       string
	Username     string
	IsDM         bool
	IsGroup      bool
	IsReplyToBot bool
	MentionsBot  bool
	Text         string
	RawText      string
	CreatedAt    time.Time
}

type Policy struct {
	RespondInDM     bool
	RespondInGroups string
}

type Status struct {
	Name    string `json:"name"`
	Enabled bool   `json:"enabled"`
	Running bool   `json:"running"`
	Error   string `json:"error,omitempty"`
}
