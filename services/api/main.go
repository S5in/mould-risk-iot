// Command api is the REST API for the dashboard.
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

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/S5in/mould-risk-iot/internal/apistore"
	"github.com/S5in/mould-risk-iot/internal/auth"
	"github.com/S5in/mould-risk-iot/internal/httpapi"
)

func main() {
	log := slog.New(slog.NewTextHandler(os.Stdout, nil))

	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		log.Error("DATABASE_URL is not set")
		os.Exit(1)
	}
	tokens, err := auth.NewTokens(os.Getenv("JWT_SECRET"), time.Hour)
	if err != nil {
		log.Error("JWT_SECRET problem", "err", err)
		os.Exit(1)
	}
	addr := os.Getenv("API_ADDR")
	if addr == "" {
		addr = ":8080"
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	pool, err := pgxpool.New(ctx, dbURL)
	if err != nil {
		log.Error("database config invalid", "err", err)
		os.Exit(1)
	}
	defer pool.Close()

	store := apistore.New(pool)
	var handler http.Handler = httpapi.New(store, tokens, log).Handler()

	// TEMPORARY until login exists (step 5): act as one fixed organisation.
	if slug := os.Getenv("DEV_ORG_SLUG"); slug != "" {
		orgID, err := store.OrgIDBySlug(ctx, slug)
		if err != nil {
			log.Error("DEV_ORG_SLUG not found", "slug", slug, "err", err)
			os.Exit(1)
		}
		log.Warn("DEV MODE: every request is treated as this organisation", "slug", slug)
		handler = httpapi.DevOrg(orgID, handler)
	}

	srv := &http.Server{
		Addr:              addr,
		Handler:           handler,
		ReadHeaderTimeout: 5 * time.Second,
	}

	go func() {
		<-ctx.Done()
		shutCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = srv.Shutdown(shutCtx)
	}()

	log.Info("api listening", "addr", addr)
	if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		log.Error("server stopped", "err", err)
		os.Exit(1)
	}
}
