package integration_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"strings"
	"testing"

	"github.com/gharkness/go-gql-demo/internal/auth"
	"github.com/gharkness/go-gql-demo/internal/reviews"
)

const createReview = `mutation { createReview(input:{bookId:"1",body:"Helpful example",rating:4}) { id body author { id } } }`

func graphqlRequest(t *testing.T, endpoint, document, token string) map[string]any {
	t.Helper()
	b, _ := json.Marshal(map[string]string{"query": document})
	req, _ := http.NewRequest("POST", endpoint, strings.NewReader(string(b)))
	req.Header.Set("Content-Type", "application/json")
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	var result map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		t.Fatal(err)
	}
	return result
}

func TestAnonymousCannotCreateReview(t *testing.T) {
	s := httptest.NewServer(reviews.Handler(database(t), nil))
	defer s.Close()
	result := graphqlRequest(t, s.URL, createReview, "")
	errs, ok := result["errors"].([]any)
	if !ok || errs[0].(map[string]any)["extensions"].(map[string]any)["code"] != "UNAUTHENTICATED" {
		t.Fatalf("expected auth rejection: %v", result)
	}
}
func TestAuthenticatedMutationPersistsAndUsesTokenIdentity(t *testing.T) {
	db := database(t)
	s := httptest.NewServer(reviews.Handler(db, auth.New("http://localhost:8085/realms/bookstore", "http://localhost:8085/realms/bookstore/protocol/openid-connect/certs").Middleware))
	defer s.Close()
	result := graphqlRequest(t, s.URL, createReview, token(t))
	if result["errors"] != nil {
		t.Fatalf("mutation: %v", result)
	}
	review := result["data"].(map[string]any)["createReview"].(map[string]any)
	// A new handler / database connection verifies persistence, without querying SQL as a side channel.
	other := httptest.NewServer(reviews.Handler(database(t), nil))
	defer other.Close()
	saved := query(t, other.URL, `{ review(id:"`+review["id"].(string)+`") { body author { id } createdAt } }`)["review"].(map[string]any)
	if saved["body"] != "Helpful example" || saved["author"].(map[string]any)["id"] != "1" || saved["createdAt"] == "" {
		t.Fatalf("persisted review: %v", saved)
	}
}
func TestKeycloakDisablesPasswordGrant(t *testing.T) {
	if os.Getenv("INTEGRATION") != "1" {
		t.Skip("requires Keycloak")
	}
	resp, err := http.PostForm("http://localhost:8085/realms/bookstore/protocol/openid-connect/token", url.Values{"grant_type": {"password"}, "client_id": {"bookstore"}, "username": {"ada"}, "password": {"ada"}})
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusBadRequest && resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("password grant must be disabled; got HTTP %d", resp.StatusCode)
	}
}
