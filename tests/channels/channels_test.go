package channels_test

import (
	"testing"
	"time"

	"vietclaw/internal/channels"
)

func TestShouldHandle(t *testing.T) {
	policy := channels.Policy{RespondInDM: true}
	if !channels.ShouldHandle(channels.InboundMessage{IsDM: true}, policy) {
		t.Fatal("DM should be handled")
	}
	if !channels.ShouldHandle(channels.InboundMessage{IsGroup: true, MentionsBot: true}, policy) {
		t.Fatal("group mention should be handled")
	}
	if !channels.ShouldHandle(channels.InboundMessage{IsGroup: true, IsReplyToBot: true}, policy) {
		t.Fatal("group reply should be handled")
	}
	if channels.ShouldHandle(channels.InboundMessage{IsGroup: true}, policy) {
		t.Fatal("plain group message should be ignored")
	}
	if !channels.ShouldHandle(channels.InboundMessage{IsGroup: true}, channels.Policy{RespondInGroups: "always"}) {
		t.Fatal("always group policy should handle plain group message")
	}
	if channels.ShouldHandle(channels.InboundMessage{IsGroup: true, MentionsBot: true}, channels.Policy{RespondInGroups: "never"}) {
		t.Fatal("never group policy should ignore group message")
	}
}

func TestStripMentions(t *testing.T) {
	if got := channels.StripMentions("<@123> deploy đi", []string{"<@123>", "<@!123>"}); got != "deploy đi" {
		t.Fatalf("discord mention strip = %q", got)
	}
	if got := channels.StripMentions("@vietclaw_bot hỏi gì đó", []string{"@vietclaw_bot"}); got != "hỏi gì đó" {
		t.Fatalf("telegram mention strip = %q", got)
	}
}

func TestSessionKey(t *testing.T) {
	tests := map[string]channels.InboundMessage{
		"discord:dm:u1":               {Platform: "discord", IsDM: true, UserID: "u1"},
		"discord:guild:g1:channel:c1": {Platform: "discord", GuildID: "g1", ChannelID: "c1"},
		"discord:guild:g1:thread:t1":  {Platform: "discord", GuildID: "g1", ChannelID: "c1", ThreadID: "t1"},
		"telegram:private:u1":         {Platform: "telegram", IsDM: true, UserID: "u1"},
		"telegram:chat:c1":            {Platform: "telegram", ChatID: "c1"},
		"telegram:chat:c1:topic:t1":   {Platform: "telegram", ChatID: "c1", ThreadID: "t1"},
	}
	for want, msg := range tests {
		if got := channels.SessionKey(msg); got != want {
			t.Fatalf("SessionKey = %q, want %q", got, want)
		}
	}
}

func TestTTLGuard(t *testing.T) {
	guard := channels.NewTTLGuard(time.Minute)
	if !guard.Seen("discord:1") {
		t.Fatal("first seen should pass")
	}
	if guard.Seen("discord:1") {
		t.Fatal("duplicate should be blocked")
	}
}
