package entry

import (
	"context"
	"os"
	"os/signal"
	"syscall"

	"golang.org/x/exp/slog"
)

type Application interface {
	Log() *slog.Logger
	Fail(message string, err error)
	Stop()
}

func NewApplication(name string) (Application, context.Context) {
	// Prepare a logger that we can write structured log messages to
	pid := os.Getpid()
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil)).With(
		"pid", pid,
		"application", name,
	)
	logger.Info("Process starting")

	// Shut down cleanly on signal
	ctx, close := signal.NotifyContext(context.Background(), os.Interrupt, os.Kill, syscall.SIGTERM)

	return &application{
		closeCtx: close,
		logger:   logger,
	}, ctx
}

type application struct {
	closeCtx context.CancelFunc
	logger   *slog.Logger
}

func (a *application) Log() *slog.Logger {
	return a.logger
}

func (a *application) Fail(message string, err error) {
	a.logger.Error(message, "error", err)
	os.Exit(1)
}

func (a *application) Stop() {
	a.logger.Info("Process stopping")
	a.closeCtx()
}
