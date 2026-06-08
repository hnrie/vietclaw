package main

import (
	"context"
	"fmt"
	"strings"

	"vietclaw/internal/memory"
)

const (
	memoryScopeLocal      = "user:local"
	defaultMemoryCLILimit = 100
	defaultSearchCLILimit = 20
)

func runMemory(args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("memory command is required: list|add|search")
	}
	service, cleanup, err := localAgent()
	if err != nil {
		return err
	}
	defer cleanup()

	ctx := context.Background()
	switch args[0] {
	case "list":
		records, err := service.Memory().List(ctx, memoryScopeLocal, defaultMemoryCLILimit)
		if err != nil {
			return err
		}
		for _, rec := range records {
			fmt.Printf("%d [%s] %s\n", rec.ID, rec.Kind, rec.Content)
		}
		return nil
	case "add":
		if len(args) < 2 {
			return fmt.Errorf("memory add content is required")
		}
		rec, err := service.Memory().Add(ctx, memoryRecord(strings.Join(args[1:], " ")))
		if err != nil {
			return err
		}
		fmt.Printf("ok, t lưu: %s\n", rec.Content)
		return nil
	case "search":
		if len(args) < 2 {
			return fmt.Errorf("memory search query is required")
		}
		records, err := service.Memory().Search(ctx, memoryScopeLocal, strings.Join(args[1:], " "), defaultSearchCLILimit)
		if err != nil {
			return err
		}
		for _, rec := range records {
			fmt.Printf("%d [%s] %s\n", rec.ID, rec.Kind, rec.Content)
		}
		return nil
	default:
		return fmt.Errorf("unknown memory command %q", args[0])
	}
}

func memoryRecord(content string) memory.Record {
	return memory.Record{
		Scope:      memoryScopeLocal,
		Kind:       memory.KindNote,
		Content:    content,
		Confidence: memory.ConfidenceConfirmed,
	}
}
