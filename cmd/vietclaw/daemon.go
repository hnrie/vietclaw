package main

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"vietclaw/internal/agent"
	"vietclaw/internal/app"
	"vietclaw/internal/channels"
	discordchannel "vietclaw/internal/channels/discord"
	telegramchannel "vietclaw/internal/channels/telegram"
	"vietclaw/internal/config"
	"vietclaw/internal/db"
	"vietclaw/internal/logging"
	"vietclaw/internal/version"
	webserver "vietclaw/internal/web"
)

const (
	defaultReadHeaderTimeout = 5 * time.Second
	defaultShutdownTimeout   = 5 * time.Second
)

func runDaemon() error {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	paths, cfg, err := loadOrCreateConfig()
	if err != nil {
		return err
	}

	logger, logFile, err := logging.New(paths.LogFile)
	if err != nil {
		return err
	}
	defer logFile.Close()

	database, err := db.Open(cfg.Database.Path)
	if err != nil {
		return err
	}
	defer database.Close()

	if err := db.ApplySchema(database); err != nil {
		return err
	}
	if err := os.MkdirAll(cfg.Agent.Workspace, 0o755); err != nil {
		return fmt.Errorf("create workspace: %w", err)
	}

	application := &app.App{
		Config:    cfg,
		DB:        database,
		Logger:    logger,
		StartTime: time.Now(),
		Version:   version.Current(),
		DataDir:   paths.DataDir,
		LogFile:   paths.LogFile,
	}
	application.Agent = agent.NewService(cfg, database)
	application.Channels = newChannelManager(cfg, application.Agent, database, logger)

	addr := fmt.Sprintf("%s:%d", cfg.Server.Host, cfg.Server.Port)
	logger.Printf("daemon starting host=%s port=%d data_dir=%s", cfg.Server.Host, cfg.Server.Port, paths.DataDir)
	if application.Channels != nil {
		application.Channels.Start(ctx)
	}

	server := &http.Server{
		Addr:              addr,
		Handler:           webserver.NewRouter(application),
		ReadHeaderTimeout: defaultReadHeaderTimeout,
	}
	return serveUntilStopped(ctx, server, logger)
}

func serveUntilStopped(ctx context.Context, server *http.Server, logger *log.Logger) error {
	errCh := make(chan error, 1)
	go func() {
		if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			errCh <- err
			return
		}
		errCh <- nil
	}()

	select {
	case err := <-errCh:
		return err
	case <-ctx.Done():
		shutdownCtx, cancel := context.WithTimeout(context.Background(), defaultShutdownTimeout)
		defer cancel()
		logger.Printf("daemon shutting down")
		return server.Shutdown(shutdownCtx)
	}
}

func newChannelManager(cfg config.Config, service *agent.Service, database *sql.DB, logger *log.Logger) *channels.Manager {
	handler := channels.NewHandler(service, database, logger)
	adapters := []channels.Adapter{}
	if cfg.Channels.Discord.Enabled {
		adapters = append(adapters, discordchannel.New(cfg.Channels.Discord, handler))
	}
	if cfg.Channels.Telegram.Enabled {
		adapters = append(adapters, telegramchannel.New(cfg.Channels.Telegram, handler))
	}
	return channels.NewManager(cfg, logger, adapters)
}
