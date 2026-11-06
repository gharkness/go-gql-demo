package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/99designs/gqlgen/graphql/playground"
	"github.com/gharkness/go-gql-demo/internal/accounts"
	"github.com/gharkness/go-gql-demo/internal/auth"
	"github.com/gharkness/go-gql-demo/internal/catalog"
	"github.com/gharkness/go-gql-demo/internal/reviews"
	"github.com/gharkness/go-gql-demo/internal/store"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	if err := run(ctx); err != nil {
		slog.Error("subgraph stopped", "error", err)
		os.Exit(1)
	}
}
func run(ctx context.Context) error {
	db, err := store.Open(os.Getenv("MYSQL_DSN"))
	if err != nil {
		return fmt.Errorf("connect MySQL: %w", err)
	}
	defer db.Close()
	var h http.Handler
	switch os.Getenv("SUBGRAPH") {
	case "accounts":
		h = accounts.Handler(db)
	case "catalog":
		h = catalog.Handler(db)
	case "reviews":
		issuer, jwks := os.Getenv("OIDC_ISSUER"), os.Getenv("OIDC_JWKS_URL")
		if issuer == "" || jwks == "" {
			return errors.New("Reviews requires OIDC_ISSUER and OIDC_JWKS_URL")
		}
		h = reviews.Handler(db, auth.New(issuer, jwks).Middleware)
	default:
		return errors.New("SUBGRAPH must be accounts, catalog, or reviews")
	}
	mux := http.NewServeMux()
	mux.Handle("/graphql", h)
	mux.Handle("/", playground.Handler("Bookstore subgraph", "/graphql"))
	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, r *http.Request) {
		check, cancel := context.WithTimeout(r.Context(), 2*time.Second)
		defer cancel()
		if err := db.PingContext(check); err != nil {
			http.Error(w, "database unavailable", http.StatusServiceUnavailable)
			return
		}
		fmt.Fprintln(w, "ok")
	})
	address := os.Getenv("LISTEN_ADDR")
	if address == "" {
		address = ":8080"
	}
	srv := &http.Server{Addr: address, Handler: mux, ReadHeaderTimeout: 5 * time.Second, IdleTimeout: 60 * time.Second, BaseContext: func(net.Listener) context.Context { return ctx }}
	stopped := make(chan error, 1)
	go func() { stopped <- srv.ListenAndServe() }()
	slog.Info("subgraph listening", "subgraph", os.Getenv("SUBGRAPH"), "address", address)
	select {
	case err := <-stopped:
		return err
	case <-ctx.Done():
		shutdown, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := srv.Shutdown(shutdown); err != nil {
			srv.Close()
			return err
		}
		return nil
	}
}
