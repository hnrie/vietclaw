package main

import (
	"context"
	"fmt"
	"strings"

	"vietclaw/internal/agent"
)

const cliChannel = "cli"

func runChat(args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("chat message is required")
	}
	service, cleanup, err := localAgent()
	if err != nil {
		return err
	}
	defer cleanup()

	resp, err := service.Chat(context.Background(), agent.ChatRequest{
		UserID:  agent.DefaultUserID,
		Channel: cliChannel,
		Message: strings.Join(args, " "),
	})
	if err != nil {
		return err
	}
	fmt.Println(resp.Reply)
	return nil
}
