// Command login obtains a short-lived token using Authorization Code + PKCE.
// Credentials are entered only at Keycloak, never into this command.
package main

import (
	"context"
	"crypto/rand"
	"crypto/subtle"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"time"

	"github.com/coreos/go-oidc/v3/oidc"
	"golang.org/x/oauth2"
)

func main() {
	if err := run(); err != nil {
		log.Fatal(err)
	}
}
func run() error {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
	defer cancel()
	const issuer = "http://localhost:8085/realms/bookstore"
	provider, err := oidc.NewProvider(ctx, issuer)
	if err != nil {
		return fmt.Errorf("discover Keycloak: %w", err)
	}
	config := oauth2.Config{ClientID: "bookstore", RedirectURL: "http://localhost:8083/", Endpoint: provider.Endpoint(), Scopes: []string{oidc.ScopeOpenID}}
	config.Endpoint.AuthStyle = oauth2.AuthStyleInParams
	state, nonce, verifier := rand.Text(), rand.Text(), oauth2.GenerateVerifier()
	listener, err := net.Listen("tcp", "127.0.0.1:8083")
	if err != nil {
		return fmt.Errorf("listen for login callback: %w", err)
	}
	tokens := make(chan string, 1)
	mux := http.NewServeMux()
	mux.HandleFunc("GET /{$}", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Cache-Control", "no-store")
		w.Header().Set("Referrer-Policy", "no-referrer")
		w.Header().Set("Content-Security-Policy", "default-src 'none'; frame-ancestors 'none'")
		q := r.URL.Query()
		if subtle.ConstantTimeCompare([]byte(q.Get("state")), []byte(state)) != 1 || q.Get("iss") != issuer {
			http.Error(w, "Invalid login response", 400)
			return
		}
		if q.Get("error") != "" {
			http.Error(w, "Login was declined. Retry the command.", 400)
			return
		}
		token, err := config.Exchange(ctx, q.Get("code"), oauth2.VerifierOption(verifier))
		if err != nil {
			http.Error(w, "Code exchange failed. Retry the command.", 400)
			return
		}
		rawID, ok := token.Extra("id_token").(string)
		if !ok {
			http.Error(w, "Missing identity token", 400)
			return
		}
		id, err := provider.Verifier(&oidc.Config{ClientID: "bookstore"}).Verify(ctx, rawID)
		if err != nil || id.Nonce != nonce {
			http.Error(w, "Invalid identity token", 400)
			return
		}
		fmt.Fprintln(w, "Signed in. Return to your terminal; you can close this tab.")
		select {
		case tokens <- token.AccessToken:
		default:
		}
	})
	srv := &http.Server{Handler: mux, ReadHeaderTimeout: 5 * time.Second, IdleTimeout: 30 * time.Second}
	failures := make(chan error, 1)
	go func() {
		if err := srv.Serve(listener); err != nil && err != http.ErrServerClosed {
			failures <- err
		}
	}()
	defer func() {
		shutdown, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		srv.Shutdown(shutdown)
	}()
	fmt.Fprintln(os.Stderr, "Open this URL in your browser and sign in as ada / ada (local demo only):")
	fmt.Fprintln(os.Stderr, config.AuthCodeURL(state, oauth2.S256ChallengeOption(verifier), oidc.Nonce(nonce)))
	select {
	case token := <-tokens:
		fmt.Println(token)
		return nil
	case err := <-failures:
		return err
	case <-ctx.Done():
		return fmt.Errorf("login timed out: %w", ctx.Err())
	}
}
