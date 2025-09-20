package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"wbtest/internal/app"
	"wbtest/internal/config"
	"wbtest/pkg/logger"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to load configuration: %v\n", err)
		os.Exit(1)
	}

	log, err := logger.NewAdapter(cfg)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to initialize logger: %v\n", err)
		os.Exit(1)
	}

	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	log.Infow("application starting", "env", cfg.Env)

	err = app.Run(ctx, cfg, log)
	if err != nil {
		log.Errorw("application failed", "error", err)
		cancel()
	}

	log.Infow("application exited normally")
}
