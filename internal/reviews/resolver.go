package reviews

import (
	"context"
	"database/sql"
	"errors"

	"github.com/gharkness/go-gql-demo/internal/store"
)

type Resolver struct {
	DB     *store.Store
	Events *Broker
}

func scanReview(row interface{ Scan(...any) error }) (*Review, error) {
	r := &Review{Book: &Book{}, Author: &User{}}
	err := row.Scan(&r.ID, &r.Body, &r.Rating, &r.CreatedAt, &r.Book.ID, &r.Author.ID)
	return r, err
}
func (r *Resolver) review(ctx context.Context, id string) (*Review, error) {
	review, err := scanReview(r.DB.QueryRowContext(ctx, "SELECT id,body,rating,created_at,book_id,author_id FROM reviews WHERE id=?", id))
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	return review, err
}
func (r *Resolver) reviews(ctx context.Context, id string, first, offset int) ([]*Review, error) {
	if first < 1 || first > 100 || offset < 0 {
		return nil, codedError("BAD_USER_INPUT", "first must be 1..100 and offset must be nonnegative")
	}
	rows, err := r.DB.QueryContext(ctx, "SELECT id,body,rating,created_at,book_id,author_id FROM reviews WHERE book_id=? ORDER BY created_at,id LIMIT ? OFFSET ?", id, first, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	result := []*Review{}
	for rows.Next() {
		review, err := scanReview(rows)
		if err != nil {
			return nil, err
		}
		result = append(result, review)
	}
	return result, rows.Err()
}
