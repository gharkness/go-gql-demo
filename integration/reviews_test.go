package integration_test

import (
	"net/http/httptest"
	"testing"

	"github.com/gharkness/go-gql-demo/internal/reviews"
)

func TestReviewsResolvesBookReviews(t *testing.T) {
	s := httptest.NewServer(reviews.Handler(database(t), nil))
	defer s.Close()
	data := query(t, s.URL, `{ _entities(representations:[{__typename:"Book",id:"1"}]) { ... on Book { reviews { body rating author { id } } } } }`)
	review := data["_entities"].([]any)[0].(map[string]any)["reviews"].([]any)[0].(map[string]any)
	if review["body"] != "A pioneering read." || review["rating"] != float64(5) {
		t.Fatalf("review: %v", review)
	}
}
func TestReviewsPaginationIsBounded(t *testing.T) {
	s := httptest.NewServer(reviews.Handler(database(t), nil))
	defer s.Close()
	data := query(t, s.URL, `{ _entities(representations:[{__typename:"Book",id:"1"}]) { ... on Book { reviews(first:1,offset:0) { id } } } }`)
	if len(data["_entities"].([]any)[0].(map[string]any)["reviews"].([]any)) != 1 {
		t.Fatalf("pagination: %v", data)
	}
}
