package integration_test

import (
	"os"
	"testing"
)

func gateway(t *testing.T) string {
	t.Helper()
	if os.Getenv("E2E") != "1" {
		t.Skip("run make test-e2e with full stack running")
	}
	return "http://localhost:8080/graphql"
}
func TestGatewayJoinsCatalogAndAccounts(t *testing.T) {
	data := query(t, gateway(t), `query { book(id:"1") { title author { name username } } }`)
	book := data["book"].(map[string]any)
	if book["title"] != "Computing Notes" || book["author"].(map[string]any)["name"] != "Ada Lovelace" {
		t.Fatalf("joined book: %v", book)
	}
}
func TestGatewayJoinsAllThreeSubgraphs(t *testing.T) {
	data := query(t, gateway(t), `{ book(id:"1") { title author { name } reviews(first: 5) { body author { name } book { title } } } }`)
	book := data["book"].(map[string]any)
	review := book["reviews"].([]any)[0].(map[string]any)
	if review["author"].(map[string]any)["name"] != "Grace Hopper" || review["book"].(map[string]any)["title"] != "Computing Notes" {
		t.Fatalf("three-way join: %v", book)
	}
}
func TestGatewayAuthenticatedMutation(t *testing.T) {
	result := graphqlRequest(t, gateway(t), createReview, token(t))
	if result["errors"] != nil {
		t.Fatalf("mutation: %v", result)
	}
	if result["data"].(map[string]any)["createReview"].(map[string]any)["author"].(map[string]any)["id"] != "1" {
		t.Fatalf("wrong author: %v", result)
	}
}
func TestGatewaySubscription(t *testing.T) {
	endpoint := gateway(t)
	subscriptionRoundTrip(t, endpoint, endpoint)
}
