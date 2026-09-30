package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"psycho/config"
	"psycho/modules/server"
	"psycho/zlogger"
)

func main() {
	cfg, err := config.Load("config/config.yaml")
	if err != nil {
		log.Fatalf("load config: %v", err)
	}

	logger := zlogger.New(cfg.Middleware.Logger.Level)

	handler, err := server.NewHandler(cfg, logger)
	if err != nil {
		log.Fatalf("setup server: %v", err)
	}

	srv := &http.Server{
		Addr:         fmt.Sprintf(":%s", cfg.App.HTTP.Port),
		Handler:      handler,
		ReadTimeout:  time.Duration(cfg.App.HTTP.ReadTimeoutInSec) * time.Second,
		WriteTimeout: time.Duration(cfg.App.HTTP.WriteTimeoutInSec) * time.Second,
		IdleTimeout:  time.Duration(cfg.App.HTTP.IdleTimeoutInSec) * time.Second,
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	errCh := make(chan error, 1)
	go func() {
		errCh <- srv.ListenAndServe()
	}()

	logger.Info(context.Background(), "starting HTTP server", zlogger.Field{Key: "addr", Value: srv.Addr})

	select {
	case err := <-errCh:
		if err != nil && !errors.Is(err, http.ErrServerClosed) {
			logger.Error(context.Background(), "server error", zlogger.Field{Key: "error", Value: err.Error()})
		}
	case <-ctx.Done():
	}

	// Graceful shutdown: SIGINT/SIGTERM (e.g. podman stop) finishes
	// in-flight analyses instead of dropping them.
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := srv.Shutdown(shutdownCtx); err != nil {
		logger.Error(context.Background(), "shutdown error", zlogger.Field{Key: "error", Value: err.Error()})
	}
}
