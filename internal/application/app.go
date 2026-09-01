package application

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"

	httpadapter "go_print_report_from_dataset/internal/adapters/http"
	"go_print_report_from_dataset/internal/domain/ports"
)

var appLogger = slog.New(slog.NewTextHandler(os.Stderr, nil)).With("tool", "print-report-from-dataset")

type App struct {
	config   Config
	parser   ports.DatasetParser
	renderer ports.PreviewRenderer
	server   *http.Server
}

func New(config Config, parser ports.DatasetParser, renderer ports.PreviewRenderer, server *http.Server) *App {
	return &App{
		config:   config,
		parser:   parser,
		renderer: renderer,
		server:   server,
	}
}

func (a *App) Run(ctx context.Context) error {
	if a.server == nil {
		handler := httpadapter.NewHandler(a.parser, a.renderer, a.config.MaxInputBytes)
		a.server = &http.Server{
			Addr:         a.config.Addr,
			Handler:      handler.Routes(),
			ReadTimeout:  a.config.ReadTimeout,
			WriteTimeout: a.config.WriteTimeout,
		}
	}

	errc := make(chan error, 1)
	go func() {
		appLogger.Info("start preview server", "addr", a.server.Addr)
		if err := a.server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			errc <- fmt.Errorf("start server: %w", err)
			return
		}
		errc <- nil
	}()

	select {
	case <-ctx.Done():
		shutdownCtx, cancel := context.WithTimeout(context.Background(), a.config.ShutdownTimeout)
		defer cancel()
		if err := a.server.Shutdown(shutdownCtx); err != nil {
			return fmt.Errorf("shutdown server: %w", err)
		}
		if err := <-errc; err != nil {
			return err
		}
		return ctx.Err()
	case err := <-errc:
		return err
	}
}
