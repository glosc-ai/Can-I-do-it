package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"time"

	"github.com/gloscai/template-go-vue3-docker/server/analysis"
	"github.com/gloscai/template-go-vue3-docker/server/cache"
	"github.com/gloscai/template-go-vue3-docker/server/config"
	"github.com/gloscai/template-go-vue3-docker/server/database"
	"github.com/gloscai/template-go-vue3-docker/server/health"
	"github.com/gloscai/template-go-vue3-docker/server/plans"
	"github.com/gloscai/template-go-vue3-docker/server/settings"
	"github.com/gloscai/template-go-vue3-docker/server/tasks"
	"github.com/gloscai/template-go-vue3-docker/server/users"
)

func run(ctx context.Context) error {
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("loading configuration: %w", err)
	}

	logger := newLogger(cfg.LogLevel)
	db, err := database.Open(ctx, cfg.Database)
	if err != nil {
		return err
	}
	defer db.Close()

	if cfg.Database.AutoMigrate {
		if err := database.Migrate(ctx, db, cfg.Database.Driver); err != nil {
			return err
		}
	}

	redisClient, err := cache.Open(ctx, cfg.Redis)
	if err != nil {
		return err
	}
	defer redisClient.Close()

	mux := http.NewServeMux()
	health.New(db, redisClient).Register(mux)
	tasks.NewHandler(tasks.NewSQLStore(db, cfg.Database.Driver)).Register(mux)
	userService := users.New(db, users.Config{Driver: cfg.Database.Driver, DiscoveryURL: cfg.SSO.DiscoveryURL, ClientID: cfg.SSO.ClientID, ClientSecret: cfg.SSO.ClientSecret, RedirectURI: cfg.SSO.RedirectURI, CookieName: cfg.Session.CookieName, TTL: cfg.Session.TTL, Secure: cfg.Session.Secure})
	userService.Register(mux)
	plans.New(db, cfg.Database.Driver, cfg.Storage.Directory, cfg.Storage.MaxUploadBytes).Register(mux)
	settings.New(db, cfg.Database.Driver, cfg.EncryptionKey).Register(mux)
	go analysis.NewWorker(db, cfg.Database.Driver, cfg.EncryptionKey).Run(ctx)

	handler := withRecovery(logger,
		withCORS(cfg.CORSOrigins,
			withRequestLog(logger,
				withRequestID(userService.Middleware(mux)),
			),
		),
	)

	srv := &http.Server{
		Addr:              cfg.HTTPAddr,
		Handler:           handler,
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       10 * time.Second,
		WriteTimeout:      30 * time.Second,
		IdleTimeout:       120 * time.Second,
	}

	errCh := make(chan error, 1)
	go func() {
		logger.Info("server started", "addr", cfg.HTTPAddr, "env", cfg.Environment)
		errCh <- srv.ListenAndServe()
	}()

	select {
	case err := <-errCh:
		if errors.Is(err, http.ErrServerClosed) {
			return nil
		}
		return fmt.Errorf("serving HTTP: %w", err)
	case <-ctx.Done():
		logger.Info("server shutting down")
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		if err := srv.Shutdown(shutdownCtx); err != nil {
			return fmt.Errorf("shutting down HTTP server: %w", err)
		}
		return nil
	}
}

func newLogger(levelName string) *slog.Logger {
	var level slog.Level
	if err := level.UnmarshalText([]byte(levelName)); err != nil {
		level = slog.LevelInfo
	}
	return slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: level}))
}
