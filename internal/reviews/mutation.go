package reviews

import (
	"context"
	"database/sql"
	"errors"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/gharkness/go-gql-demo/internal/auth"
	"github.com/google/uuid"
	"github.com/vektah/gqlparser/v2/gqlerror"
)

func codedError(code, message string) error {
	return &gqlerror.Error{Message: message, Extensions: map[string]any{"code": code}}
}
func (r *Resolver) createReview(ctx context.Context, input CreateReviewInput) (*Review, error) {
	identity, ok := auth.FromContext(ctx)
	if !ok || !time.Now().Before(identity.ExpiresAt) {
		return nil, codedError("UNAUTHENTICATED", "Sign in to create a review")
	}
	body := strings.TrimSpace(input.Body)
	if input.Rating < 1 || input.Rating > 5 || utf8.RuneCountInString(body) < 1 || utf8.RuneCountInString(body) > 2000 {
		return nil, codedError("BAD_USER_INPUT", "rating must be 1..5 and body must contain 1..2000 characters")
	}
	var authorID string
	err := r.DB.QueryRowContext(ctx, "SELECT id FROM users WHERE oidc_subject=?", identity.Subject).Scan(&authorID)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, codedError("FORBIDDEN", "This account has no bookstore profile")
	}
	if err != nil {
		return nil, err
	}
	var exists bool
	if err := r.DB.QueryRowContext(ctx, "SELECT EXISTS(SELECT 1 FROM books WHERE id=?)", input.BookID).Scan(&exists); err != nil {
		return nil, err
	}
	if !exists {
		return nil, codedError("BAD_USER_INPUT", "Book does not exist")
	}
	review := &Review{ID: uuid.NewString(), Body: body, Rating: input.Rating, CreatedAt: time.Now().UTC().Truncate(time.Microsecond), Author: &User{ID: authorID}, Book: &Book{ID: input.BookID}}
	_, err = r.DB.ExecContext(ctx, "INSERT INTO reviews (id,book_id,author_id,body,rating,created_at) VALUES (?,?,?,?,?,?)", review.ID, review.Book.ID, review.Author.ID, review.Body, review.Rating, review.CreatedAt)
	if err != nil {
		return nil, err
	}
	r.Events.Publish(review)
	return review, nil
}
