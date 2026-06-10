package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strings"

	"vietclaw/internal/config"
	"vietclaw/internal/db"
	"vietclaw/internal/harness"
)

func runHarness(args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("harness command is required: create|run|list|show|diff|cancel|cleanup")
	}
	_, cfg, database, cleanup, err := localDatabase()
	if err != nil {
		return err
	}
	defer cleanup()
	service := harness.New(cfg, database)
	ctx := context.Background()

	switch args[0] {
	case "create":
		if len(args) < 2 {
			return fmt.Errorf("harness create goal is required")
		}
		run, err := service.Create(ctx, harness.CreateRequest{Goal: strings.Join(args[1:], " "), WorkspaceRoot: currentDirectory()})
		if err != nil {
			return err
		}
		return printJSON(run)
	case "run":
		if len(args) < 2 {
			return fmt.Errorf("harness run goal is required")
		}
		run, err := service.Create(ctx, harness.CreateRequest{Goal: strings.Join(args[1:], " "), WorkspaceRoot: currentDirectory(), AutoRun: true})
		if err != nil {
			return err
		}
		return printJSON(run)
	case "list":
		runs, err := service.List(ctx, 20)
		if err != nil {
			return err
		}
		for _, run := range runs {
			fmt.Printf("%s [%s/%s] %s\n", run.ID, run.Status, run.Risk, run.Goal)
		}
		return nil
	case "show":
		if len(args) < 2 {
			return fmt.Errorf("harness show run_id is required")
		}
		detail, err := service.Detail(ctx, args[1])
		if errors.Is(err, sql.ErrNoRows) {
			return fmt.Errorf("harness run not found: %s", args[1])
		}
		if err != nil {
			return err
		}
		return printJSON(detail)
	case "diff":
		if len(args) < 2 {
			return fmt.Errorf("harness diff run_id is required")
		}
		diff, err := service.Diff(ctx, args[1])
		if err != nil {
			return err
		}
		fmt.Print(diff)
		return nil
	case "cancel":
		if len(args) < 2 {
			return fmt.Errorf("harness cancel run_id is required")
		}
		run, err := service.Cancel(ctx, args[1])
		if err != nil {
			return err
		}
		return printJSON(run)
	case "cleanup":
		if len(args) < 2 {
			return fmt.Errorf("harness cleanup run_id is required")
		}
		if args[1] == "--passed" || args[1] == "--failed" {
			target := strings.TrimPrefix(args[1], "--")
			runs, err := service.List(ctx, 200)
			if err != nil {
				return err
			}
			count := 0
			for _, run := range runs {
				if string(run.Status) == target {
					if _, err := service.Cleanup(ctx, run.ID); err != nil {
						return err
					}
					count++
				}
			}
			fmt.Printf("cleaned %d %s harness runs\n", count, target)
			return nil
		}
		run, err := service.Cleanup(ctx, args[1])
		if err != nil {
			return err
		}
		return printJSON(run)
	default:
		return fmt.Errorf("unknown harness command %q", args[0])
	}
}

func localDatabase() (config.Paths, config.Config, *sql.DB, func(), error) {
	paths, cfg, err := loadOrCreateConfig()
	if err != nil {
		return config.Paths{}, config.Config{}, nil, nil, err
	}
	database, err := db.Open(cfg.Database.Path)
	if err != nil {
		return config.Paths{}, config.Config{}, nil, nil, err
	}
	if err := db.ApplySchema(database); err != nil {
		_ = database.Close()
		return config.Paths{}, config.Config{}, nil, nil, err
	}
	return paths, cfg, database, func() { _ = database.Close() }, nil
}

func printJSON(value any) error {
	encoded, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return err
	}
	fmt.Println(string(encoded))
	return nil
}

func currentDirectory() string {
	wd, err := os.Getwd()
	if err != nil {
		return ""
	}
	return wd
}
