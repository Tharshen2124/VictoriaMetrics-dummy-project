package main

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/signal"
	"time"

	"github.com/gorilla/mux"
	"tharshen2124/vivmdummyproject/config"
	"tharshen2124/vivmdummyproject/db"
	"tharshen2124/vivmdummyproject/routes"
)

func main() {
	if err := run(); err != nil {
		fmt.Printf("[SERVER] Error: %v\n", err)
		os.Exit(1)
	}
}

func run() error {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()

	// Load configuration from .env file.
	cfg, err := config.LoadConfig()
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	// Connect to the Postgres database.
	if err := db.InitDB(cfg.DatabaseURL); err != nil {
		return fmt.Errorf("failed to connect to database: %w", err)
	}

	// Connect to OpenTelemetry
	otelShutdown, err := config.SetupOTelSDK(ctx)
	if err != nil {
		return fmt.Errorf("failed to set up OpenTelemetry SDK: %w", err)
	}
	defer func() {
		err = errors.Join(err, otelShutdown(context.Background()))
	}()

	// Build and configure the router.
	r := mux.NewRouter()
	routes.RegisterRoutes(r)

	srv := &http.Server{
		Addr:         ":" + cfg.Port,
		BaseContext:  func(net.Listener) context.Context { return ctx },
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 10 * time.Second,
		Handler:      r,
	}

	// Start server in a goroutine
	srvErr := make(chan error, 1)
	go func() {
		fmt.Printf("[SERVER] Starting on port %s\n", cfg.Port)
		srvErr <- srv.ListenAndServe()
	}()

	// Wait for either server error or CTRL+C
	select {
	case err = <-srvErr:
		return err
	case <-ctx.Done():
		stop()
	}

	// Gracefully shut down the server
	fmt.Println("[SERVER] Shutting down...")
	err = srv.Shutdown(context.Background())
	return err
}