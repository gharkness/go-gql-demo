package reviews

import (
	"net/http"

	"github.com/gharkness/go-gql-demo/internal/graphserver"
	"github.com/gharkness/go-gql-demo/internal/store"
)

func Handler(db *store.Store, middleware func(http.Handler) http.Handler) http.Handler {
	server := graphserver.New(NewExecutableSchema(Config{Resolvers: &Resolver{DB: db, Events: NewBroker()}}))
	if middleware != nil {
		return middleware(server)
	}
	return server
}
