package app

import (
	"context"
	"errors"
	"fmt"
	"github.com/fatalistix/crack-hash-worker/internal/config"
	"github.com/fatalistix/crack-hash-worker/internal/http/client"
	"github.com/fatalistix/crack-hash-worker/internal/http/handler"
	"github.com/fatalistix/crack-hash-worker/internal/service"
	"github.com/fatalistix/crack-hash-worker/internal/validation"
	"github.com/fatalistix/slogattr"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	slogecho "github.com/samber/slog-echo"
	"io"
	"log/slog"
	"net/http"
)

type App struct {
	e       *echo.Echo
	log     *slog.Logger
	closers []io.Closer
	port    int
}

func New(log *slog.Logger, cfg config.Config) (*App, error) {
	const op = "app.New"

	closers := make([]io.Closer, 0)

	registerer := client.NewRegisterer(log)

	id, err := registerer.Register(cfg.Manager.Address, cfg.Deployment.Port)
	if err != nil {
		log.Error("failed to register worker", slogattr.Err(err))
		return nil, fmt.Errorf("%s: register error: %w", op, err)
	}

	log.Info("worker registered", slog.String("worker id", id))

	s := service.NewCrackService(log, cfg.Manager, cfg.Worker, id)
	closers = append(closers, s)

	startHandler := handler.MakeStartTaskHandlerFunc(s)

	v, err := validation.NewRequestValidator()
	if err != nil {
		log.Error("failed to create validator", slogattr.Err(err))
		return nil, fmt.Errorf("%s: error creating request validator: %w", op, err)
	}

	e := echo.New()

	e.Validator = v

	e.POST("/internal/api/worker/hash/crack/task", startHandler)

	e.Use(slogecho.New(log))
	e.Use(middleware.Recover())
	e.Use(middleware.RequestID())

	return &App{
		e:       e,
		log:     log,
		closers: closers,
		port:    cfg.Deployment.Port,
	}, nil
}

func (a *App) Start() error {
	const op = "app.Start"

	log := a.log.With(
		slog.String("op", op),
	)

	log.Info("starting server", slog.Int("port", a.port))

	address := fmt.Sprintf(":%d", a.port)

	if err := a.e.Start(address); err != nil && !errors.Is(err, http.ErrServerClosed) {
		log.Error("error starting http server", slog.Int("port", a.port), slogattr.Err(err))
		return fmt.Errorf("%s: error starting http server: %w", op, err)
	}

	log.Info("server stopped", slog.Int("port", a.port))

	return nil
}

func (a *App) Stop(ctx context.Context) error {
	const op = "app.Stop"

	log := a.log.With(
		slog.String("op", op),
	)

	defer a.close()

	log.Info("stopping server", slog.Int("port", a.port))

	if err := a.e.Shutdown(ctx); err != nil {
		log.Error("error during stopping http server", slog.Int("port", a.port), slogattr.Err(err))
		return fmt.Errorf("%s: error stopping http server: %w", op, err)
	}

	a.log.Info("server stopped successfully", slog.Int("port", a.port))

	return nil
}

func (a *App) close() {
	const op = "app.close"

	log := a.log.With(
		slog.String("op", op),
	)

	log.Info("closing services...")

	for _, closer := range a.closers {
		if err := closer.Close(); err != nil {
			log.Error("error closing service", slogattr.Err(err))
		}
	}

	log.Info("all services closed")
}
