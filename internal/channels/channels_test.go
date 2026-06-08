package channels

import (
	"testing"
	"time"
)

func TestShouldHandle(t *testing.T) {
	policy := Policy{RespondInDM: true}
	if !ShouldHandle(InboundMessage{IsDM: true}, policy) {
		t.Fatal("DM should be handled")
	}
	if !ShouldHandle(InboundMessage{IsGroup: true, MentionsBot: true}, policy) {
		t.Fatal("group mention should be handled")
	}
	if !ShouldHandle(InboundMessage{IsGroup: true, IsReplyToBot: true}, policy) {
		t.Fatal("group reply should be handled")
	}
	if ShouldHandle(InboundMessage{IsGroup: true}, policy) {
		t.Fatal("plain group message should be ignored")
	}
}

func TestStripMentions(t *testing.T) {
	if got := StripMentions("<@123> deploy đi", []string{"<@123>", "<@!123>"}); got != "deploy đi" {
		t.Fatalf("discord mention strip = %q", got)
	}
	if got := StripMentions("@vietclaw_bot hỏi gì đó", []string{"@vietclaw_bot"}); got != "hỏi gì đó" {
		t.Fatalf("telegram mention strip = %q", got)
	}
}

func TestSessionKey(t *testing.T) {
	tests := map[string]InboundMessage{
		"discord:dm:u1":               {Platform: "discord", IsDM: true, UserID: "u1"},
		"discord:guild:g1:channel:c1": {Platform: "discord", GuildID: "g1", ChannelID: "c1"},
		"discord:guild:g1:thread:t1":  {Platform: "discord", GuildID: "g1", ChannelID: "c1", ThreadID: "t1"},
		"telegram:private:u1":         {Platform: "telegram", IsDM: true, UserID: "u1"},
		"telegram:chat:c1":            {Platform: "telegram", ChatID: "c1"},
		"telegram:chat:c1:topic:t1":   {Platform: "telegram", ChatID: "c1", ThreadID: "t1"},
	}
	for want, msg := range tests {
		if got := SessionKey(msg); got != want {
			t.Fatalf("SessionKey = %q, want %q", got, want)
		}
	}
}

func TestTTLGuard(t *testing.T) {
	guard := NewTTLGuard(time.Minute)
	if !guard.Seen("discord:1") {
		t.Fatal("first seen should pass")
	}
	if guard.Seen("discord:1") {
		t.Fatal("duplicate should be blocked")
	}
}
