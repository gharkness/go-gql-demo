package integration_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/gharkness/go-gql-demo/internal/accounts"
	"github.com/gharkness/go-gql-demo/internal/store"
)

func database(t *testing.T) *store.Store {
	t.Helper()
	if os.Getenv("INTEGRATION") != "1" {
		t.Skip("run make test-integration with MySQL running")
	}
	db, err := store.Open("demo:demo@tcp(127.0.0.1:3307)/bookstore?parseTime=true")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { db.Close() })
	return db
}
func query(t *testing.T, endpoint, document string) map[string]any {
	t.Helper()
	body, _ := json.Marshal(map[string]any{"query": document})
	resp, err := http.Post(endpoint, "application/json", strings.NewReader(string(body)))
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	var result map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		t.Fatal(err)
	}
	if result["errors"] != nil {
		t.Fatalf("GraphQL errors: %v", result)
	}
	return result["data"].(map[string]any)
}
func TestAccountsQueryReturnsSeededUser(t *testing.T) {
	db := database(t)
	server := httptest.NewServer(accounts.Handler(db))
	defer server.Close()
	data := query(t, server.URL, `{ user(id: "1") { id name username } }`)
	user := data["user"].(map[string]any)
	if user["name"] != "Ada Lovelace" || user["username"] != "ada" {
		t.Fatalf("unexpected user: %v", user)
	}
}
func TestAccountsResolvesFederationEntities(t *testing.T) {
	s := httptest.NewServer(accounts.Handler(database(t)))
	defer s.Close()
	data := query(t, s.URL, `{ _entities(representations:[{__typename:"User",id:"2"}]) { ... on User { name } } }`)
	entity := data["_entities"].([]any)[0].(map[string]any)
	if entity["name"] != "Grace Hopper" {
		t.Fatalf("entity: %v", entity)
	}
}
