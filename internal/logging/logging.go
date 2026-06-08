package logging

import (
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
)

func EnsureLogFile(path string) (*os.File, bool, error) {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return nil, false, fmt.Errorf("create log dir: %w", err)
	}

	_, statErr := os.Stat(path)
	file, err := os.OpenFile(path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		return nil, false, fmt.Errorf("open log file: %w", err)
	}

	return file, os.IsNotExist(statErr), nil
}

func New(path string) (*log.Logger, *os.File, error) {
	file, _, err := EnsureLogFile(path)
	if err != nil {
		return nil, nil, err
	}

	writer := io.MultiWriter(os.Stdout, file)
	logger := log.New(writer, "vietclaw ", log.LstdFlags)
	return logger, file, nil
}
