package integration_test

import (
	"context"
	"crypto/rand"
	"encoding/json"
	"html"
	"io"
	"net/http"
	"net/http/cookiejar"
	"net/http/httptest"
	"net/url"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/gharkness/go-gql-demo/internal/auth"
	"github.com/gharkness/go-gql-demo/internal/reviews"
	"golang.org/x/oauth2"
)

// token drives the real Keycloak login form as a test user. The application never
// handles a user's password: this helper is solely an integration-test browser.
func token(t *testing.T) string { t.Helper(); return loginTokens(t).AccessToken }
func loginTokens(t *testing.T) *oauth2.Token {
	t.Helper()
	const issuer = "http://localhost:8085/realms/bookstore"
	config := oauth2.Config{ClientID: "bookstore", RedirectURL: "http://localhost:8083/", Scopes: []string{"openid"}, Endpoint: oauth2.Endpoint{AuthURL: issuer + "/protocol/openid-connect/auth", TokenURL: issuer + "/protocol/openid-connect/token", AuthStyle: oauth2.AuthStyleInParams}}
	verifier := oauth2.GenerateVerifier()
	state := rand.Text()
	jar, _ := cookiejar.New(nil)
	client := &http.Client{Jar: jar, Timeout: 10 * time.Second, CheckRedirect: func(req *http.Request, via []*http.Request) error {
		if req.URL.Host == "localhost:8083" {
			return http.ErrUseLastResponse
		}
		if len(via) > 10 {
			return http.ErrUseLastResponse
		}
		return nil
	}}
	resp, err := client.Get(config.AuthCodeURL(state, oauth2.S256ChallengeOption(verifier)))
	if err != nil {
		t.Fatal(err)
	}
	body, err := io.ReadAll(resp.Body)
	resp.Body.Close()
	if err != nil {
		t.Fatal(err)
	}
	match := regexp.MustCompile(`<form[^>]+action="([^"]+)"`).FindSubmatch(body)
	if len(match) != 2 {
		t.Fatalf("Keycloak login form missing (HTTP %d)", resp.StatusCode)
	}
	action := html.UnescapeString(string(match[1]))
	actionURL, err := url.Parse(action)
	if err != nil || actionURL.Host != "localhost:8085" {
		t.Fatal("unexpected login form origin")
	}
	resp, err = client.PostForm(action, url.Values{"username": {"ada"}, "password": {"ada"}, "credentialId": {""}})
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	callback, err := resp.Location()
	if err != nil {
		t.Fatalf("login callback: %v (HTTP %d)", err, resp.StatusCode)
	}
	if callback.Query().Get("state") != state || callback.Query().Get("iss") != issuer {
		t.Fatal("authorization response state/issuer mismatch")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	access, err := config.Exchange(ctx, callback.Query().Get("code"), oauth2.VerifierOption(verifier))
	if err != nil {
		t.Fatal(err)
	}
	return access
}
func TestKeycloakRejectsMissingPKCE(t *testing.T) {
	database(t)
	client := &http.Client{Timeout: 10 * time.Second, CheckRedirect: func(req *http.Request, via []*http.Request) error { return http.ErrUseLastResponse }}
	resp, err := client.Get("http://localhost:8085/realms/bookstore/protocol/openid-connect/auth?" + url.Values{"client_id": {"bookstore"}, "redirect_uri": {"http://localhost:8083/"}, "response_type": {"code"}, "scope": {"openid"}, "state": {"test"}}.Encode())
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	location, _ := resp.Location()
	if location == nil || location.Query().Get("error") != "invalid_request" {
		t.Fatalf("missing PKCE should be rejected: HTTP %d", resp.StatusCode)
	}
}
func TestKeycloakRejectsUnregisteredRedirect(t *testing.T) {
	database(t)
	resp, err := http.Get("http://localhost:8085/realms/bookstore/protocol/openid-connect/auth?" + url.Values{"client_id": {"bookstore"}, "redirect_uri": {"http://localhost:8083/unregistered"}, "response_type": {"code"}, "scope": {"openid"}}.Encode())
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != 400 {
		t.Fatalf("unexpected redirect accepted: %d", resp.StatusCode)
	}
}
func TestInvalidSignatureCannotCreateReview(t *testing.T) {
	endpoint := gateway(t)
	parts := strings.Split(token(t), ".")
	parts[2] = "AAAA"
	result := graphqlRequest(t, endpoint, createReview, strings.Join(parts, "."))
	encoded, _ := json.Marshal(result)
	if !strings.Contains(string(encoded), "UNAUTHENTICATED") {
		t.Fatalf("forged token accepted: %s", encoded)
	}
}

func TestIDTokenCannotAuthorizeMutation(t *testing.T) {
	s := httptest.NewServer(reviews.Handler(database(t), auth.New("http://localhost:8085/realms/bookstore", "http://localhost:8085/realms/bookstore/protocol/openid-connect/certs").Middleware))
	defer s.Close()
	idToken := loginTokens(t).Extra("id_token").(string)
	result := graphqlRequest(t, s.URL, createReview, idToken)
	if result["errors"] == nil {
		t.Fatal("ID token must not be accepted as an access token")
	}
}
