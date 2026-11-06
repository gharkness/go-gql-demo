// Package graphserver configures the common GraphQL transports and limits.
package graphserver

import (
	"context"
	"log/slog"
	"net/http"
	"time"

	"github.com/99designs/gqlgen/graphql"
	"github.com/99designs/gqlgen/graphql/handler"
	"github.com/99designs/gqlgen/graphql/handler/extension"
	"github.com/99designs/gqlgen/graphql/handler/lru"
	"github.com/99designs/gqlgen/graphql/handler/transport"
	"github.com/gorilla/websocket"
	"github.com/vektah/gqlparser/v2/ast"
	"github.com/vektah/gqlparser/v2/gqlerror"
)

func New(schema graphql.ExecutableSchema) http.Handler {
	server := handler.New(schema)
	server.AddTransport(transport.Websocket{Upgrader: websocket.Upgrader{CheckOrigin: func(r *http.Request) bool {
		origin := r.Header.Get("Origin")
		return origin == "" || origin == "http://localhost:8080" || origin == "http://localhost:8081" || origin == "http://localhost:8082" || origin == "http://localhost:8084"
	}}, KeepAlivePingInterval: 15 * time.Second, InitTimeout: 5 * time.Second})
	server.AddTransport(transport.Options{})
	server.AddTransport(transport.GET{})
	server.AddTransport(transport.POST{})
	server.SetQueryCache(lru.New[*ast.QueryDocument](100))
	server.Use(extension.Introspection{}) // Introspection is intentional in this learning project.
	server.Use(extension.FixedComplexityLimit(200))
	server.SetErrorPresenter(func(ctx context.Context, err error) *gqlerror.Error {
		presented := graphql.DefaultErrorPresenter(ctx, err)
		if presented.Err != nil {
			slog.ErrorContext(ctx, "resolver failed", "error", err)
			return &gqlerror.Error{Message: "Internal server error", Path: presented.Path, Extensions: map[string]any{"code": "INTERNAL_SERVER_ERROR"}}
		}
		return presented
	})
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		r.Body = http.MaxBytesReader(w, r.Body, 1<<20)
		server.ServeHTTP(w, r)
	})
}
