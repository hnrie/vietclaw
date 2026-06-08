package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"time"

	"vietclaw/internal/config"
)

const statusHTTPTimeout = 2 * time.Second

func runStatus() error {
	_, cfg, err := loadExistingConfig()
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			fmt.Println("daemon is not running")
			return nil
		}
		return err
	}

	status, err := fetchStatus(cfg)
	if err != nil {
		fmt.Println("daemon is not running")
		return nil
	}

	encoded, err := json.MarshalIndent(status, "", "  ")
	if err != nil {
		return err
	}
	fmt.Println(string(encoded))
	return nil
}

func fetchStatus(cfg config.Config) (map[string]any, error) {
	client := &http.Client{Timeout: statusHTTPTimeout}
	url := fmt.Sprintf("http://%s:%d/status", cfg.Server.Host, cfg.Server.Port)
	req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("status endpoint returned %s", resp.Status)
	}

	var payload map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return nil, err
	}
	return payload, nil
}
