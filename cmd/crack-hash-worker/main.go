package main

import (
	"context"
	"errors"
	"github.com/fatalistix/crack-hash-worker/internal/app"
	"github.com/fatalistix/crack-hash-worker/internal/config"
	"github.com/fatalistix/slogattr"
	"github.com/golang-cz/devslog"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
)

func main() {
	log := setupLog()

	cfg := config.MustRead()

	log.Info("config loaded", slog.Any("config", cfg))

	a, err := app.New(log, cfg)
	if err != nil {
		log.Error("failed to initialize app", slogattr.Err(err))
		panic(err)
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()

	go func() {
		if err := a.Start(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Error("error starting application", slogattr.Err(err))
			panic(err)
		}
	}()

	<-ctx.Done()

	log.Info("shutting down server...", slog.Duration("shutdown timeout", cfg.Deployment.ShutdownTimeout))

	ctx, cancel := context.WithTimeout(context.Background(), cfg.Deployment.ShutdownTimeout)
	defer cancel()

	if err := a.Stop(ctx); err != nil {
		log.Error("error stopping application", slogattr.Err(err))
		panic(err)
	}
}

func setupLog() *slog.Logger {
	slogOpts := &slog.HandlerOptions{
		AddSource: true,
		Level:     slog.LevelInfo,
	}

	devSlogOpts := &devslog.Options{
		HandlerOptions: slogOpts,
	}

	return slog.New(devslog.NewHandler(os.Stdout, devSlogOpts))
}
