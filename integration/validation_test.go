package integration_test

import (
	"fmt"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gharkness/go-gql-demo/internal/auth"
	"github.com/gharkness/go-gql-demo/internal/catalog"
	"github.com/gharkness/go-gql-demo/internal/reviews"
)

func TestInvalidReviewInputIsRejected(t *testing.T) {
	s := httptest.NewServer(reviews.Handler(database(t), auth.New("http://localhost:8085/realms/bookstore", "http://localhost:8085/realms/bookstore/protocol/openid-connect/certs").Middleware))
	defer s.Close()
	access := token(t)
	for _, input := range []string{`{bookId:"1",body:"",rating:4}`, `{bookId:"1",body:"bad rating",rating:6}`, `{bookId:"missing",body:"unknown book",rating:4}`} {
		result := graphqlRequest(t, s.URL, `mutation {createReview(input:`+input+`) {id}}`, access)
		errs, ok := result["errors"].([]any)
		if !ok || errs[0].(map[string]any)["extensions"].(map[string]any)["code"] != "BAD_USER_INPUT" {
			t.Fatalf("input %s: %v", input, result)
		}
	}
}
func TestCatalogRejectsUnboundedPage(t *testing.T) {
	s := httptest.NewServer(catalog.Handler(database(t)))
	defer s.Close()
	result := graphqlRequest(t, s.URL, `{books(first:100000){id}}`, "")
	if result["errors"] == nil {
		t.Fatalf("unbounded page accepted: %v", result)
	}
}
func TestExpensiveQueryIsRejected(t *testing.T) {
	s := httptest.NewServer(catalog.Handler(database(t)))
	defer s.Close()
	var document strings.Builder
	document.WriteString("{")
	for i := 0; i < 250; i++ {
		fmt.Fprintf(&document, " b%d:book(id:\"1\"){id title} ", i)
	}
	document.WriteString("}")
	result := graphqlRequest(t, s.URL, document.String(), "")
	if result["errors"] == nil {
		t.Fatal("expensive query was accepted")
	}
}
