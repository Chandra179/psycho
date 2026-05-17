package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
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

	logger.Info(context.Background(), "starting HTTP server", zlogger.Field{Key: "addr", Value: srv.Addr})
	if err := srv.ListenAndServe(); err != nil {
		logger.Error(context.Background(), "server error", zlogger.Field{Key: "error", Value: err.Error()})
	}
}
