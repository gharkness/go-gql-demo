package store

import (
	"context"
	"database/sql"
	"errors"
	"time"

	_ "github.com/go-sql-driver/mysql"
)

type Store struct{ *sql.DB }

// User is a plain Go struct: the Go equivalent of a Java POJO.
type User struct {
	ID       string
	Username string
	Name     string
}

func Open(dsn string) (*Store, error) {
	db, err := sql.Open("mysql", dsn)
	if err != nil {
		return nil, err
	}
	db.SetMaxOpenConns(10)
	db.SetConnMaxLifetime(3 * time.Minute)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := db.PingContext(ctx); err != nil {
		db.Close()
		return nil, err
	}
	return &Store{db}, nil
}
func (s *Store) User(ctx context.Context, id string) (*User, error) {
	var u User
	err := s.QueryRowContext(ctx, "SELECT id, username, name FROM users WHERE id = ?", id).Scan(&u.ID, &u.Username, &u.Name)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	return &u, err
}
func (User) IsEntity() {}
