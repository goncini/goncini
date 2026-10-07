// Command realworld-plain serves the RealWorld "Conduit" API using only
// net/http and database/sql.
//
// Configuration comes from the environment:
//
//	DATABASE_URL  SQLite database path or file: URI (required)
//	APP_SECRET    HS256 signing key, at least 32 bytes (required)
//	ADDR          listen address (default ":8080")
package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stderr, nil))
	if err := run(logger); err != nil {
		logger.Error("fatal", "err", err)
		os.Exit(1)
	}
}

type config struct {
	addr        string
	databaseURL string
	secret      []byte
}

func loadConfig() (config, error) {
	cfg := config{
		addr:        os.Getenv("ADDR"),
		databaseURL: os.Getenv("DATABASE_URL"),
		secret:      []byte(os.Getenv("APP_SECRET")),
	}
	if cfg.addr == "" {
		cfg.addr = ":8080"
	}
	if cfg.databaseURL == "" {
		return config{}, errors.New("DATABASE_URL is required")
	}
	if len(cfg.secret) < 32 {
		return config{}, errors.New("APP_SECRET must be at least 32 bytes")
	}
	return cfg, nil
}

func run(logger *slog.Logger) error {
	cfg, err := loadConfig()
	if err != nil {
		return err
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	db, err := openDB(ctx, cfg.databaseURL)
	if err != nil {
		return err
	}
	defer db.Close()

	srv := &http.Server{
		Addr:              cfg.addr,
		Handler:           newServer(db, cfg.secret, logger),
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       15 * time.Second,
		WriteTimeout:      30 * time.Second,
		IdleTimeout:       2 * time.Minute,
		ErrorLog:          slog.NewLogLogger(logger.Handler(), slog.LevelWarn),
	}

	errc := make(chan error, 1)
	go func() {
		logger.Info("listening", "addr", cfg.addr)
		errc <- srv.ListenAndServe()
	}()

	select {
	case err := <-errc:
		return err
	case <-ctx.Done():
	}
	stop() // A second signal kills the process.

	logger.Info("shutting down")
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := srv.Shutdown(shutdownCtx); err != nil {
		return err
	}
	if err := <-errc; !errors.Is(err, http.ErrServerClosed) {
		return err
	}
	return nil
}
