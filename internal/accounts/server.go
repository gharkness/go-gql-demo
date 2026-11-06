package accounts

import (
	"net/http"

	"github.com/gharkness/go-gql-demo/internal/graphserver"
	"github.com/gharkness/go-gql-demo/internal/store"
)

func Handler(db *store.Store) http.Handler {
	return graphserver.New(NewExecutableSchema(Config{Resolvers: &Resolver{DB: db}}))
}
