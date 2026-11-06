package integration_test

import (
	"net/http/httptest"
	"testing"

	"github.com/gharkness/go-gql-demo/internal/catalog"
)

func TestCatalogReturnsBooksWithAuthorReferences(t *testing.T) {
	s := httptest.NewServer(catalog.Handler(database(t)))
	defer s.Close()
	data := query(t, s.URL, `{ books(first: 1, offset: 0) { id title genre author { id } } }`)
	books := data["books"].([]any)
	if len(books) != 1 {
		t.Fatalf("books: %v", books)
	}
	b := books[0].(map[string]any)
	if b["title"] != "Computing Notes" || b["genre"] != "TECHNOLOGY" || b["author"].(map[string]any)["id"] != "1" {
		t.Fatalf("book: %v", b)
	}
}
func TestCatalogSupportsInterfacesAndUnionFragments(t *testing.T) {
	s := httptest.NewServer(catalog.Handler(database(t)))
	defer s.Close()
	data := query(t, s.URL, `query { node(id:"1") { id ... on Book { title } } search(text:"Computing") { __typename ... on Book { title } ... on ReadingList { name books { title } } } }`)
	if data["node"].(map[string]any)["title"] != "Computing Notes" {
		t.Fatalf("node: %v", data)
	}
	hits := data["search"].([]any)
	if len(hits) != 2 || hits[0].(map[string]any)["__typename"] != "Book" || hits[1].(map[string]any)["__typename"] != "ReadingList" {
		t.Fatalf("search: %v", hits)
	}
}
