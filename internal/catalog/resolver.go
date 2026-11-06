package catalog

import (
	"context"
	"database/sql"
	"errors"
	"strings"

	"github.com/gharkness/go-gql-demo/internal/store"
	"github.com/vektah/gqlparser/v2/gqlerror"
)

type Resolver struct{ DB *store.Store }

func (r *Resolver) book(ctx context.Context, id string) (*Book, error) {
	b := &Book{Author: &User{}}
	err := r.DB.QueryRowContext(ctx, "SELECT id,title,genre,author_id FROM books WHERE id=?", id).Scan(&b.ID, &b.Title, &b.Genre, &b.Author.ID)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	return b, err
}
func (r *Resolver) books(ctx context.Context, first, offset int) ([]*Book, error) {
	if first < 1 || first > 100 || offset < 0 {
		return nil, gqlerror.Errorf("first must be 1..100 and offset must be nonnegative")
	}
	rows, err := r.DB.QueryContext(ctx, "SELECT id,title,genre,author_id FROM books ORDER BY id LIMIT ? OFFSET ?", first, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	result := []*Book{}
	for rows.Next() {
		b := &Book{Author: &User{}}
		if err := rows.Scan(&b.ID, &b.Title, &b.Genre, &b.Author.ID); err != nil {
			return nil, err
		}
		result = append(result, b)
	}
	return result, rows.Err()
}

func (r *Resolver) search(ctx context.Context, text string) ([]SearchResult, error) {
	text = strings.TrimSpace(text)
	if text == "" || len(text) > 200 {
		return nil, gqlerror.Errorf("search text must contain 1..200 bytes")
	}
	rows, err := r.DB.QueryContext(ctx, "SELECT id,title,genre,author_id FROM books WHERE LOCATE(LOWER(?),LOWER(title)) > 0 ORDER BY id LIMIT 20", text)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	hits := []SearchResult{}
	books := []*Book{}
	for rows.Next() {
		b := &Book{Author: &User{}}
		if err := rows.Scan(&b.ID, &b.Title, &b.Genre, &b.Author.ID); err != nil {
			return nil, err
		}
		hits = append(hits, b)
		books = append(books, b)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	if len(books) > 0 {
		hits = append(hits, &ReadingList{ID: "matching-books", Name: "Books matching your search", Books: books})
	}
	return hits, nil
}
