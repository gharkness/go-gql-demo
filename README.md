# Go GraphQL bookstore

A working, local Apollo Federation 2 example: three Go/gqlgen subgraphs, a **Cosmo router written in Go**, MySQL, and Keycloak. One GraphQL request can join books, users, and reviews across independently running services. No hosted Cosmo or Apollo account is needed.

## Run

Install Docker with Compose. For development, login, and tests, install Go **1.26.10 or newer**. Node is only used inside a container when regenerating federation configuration.

```sh
make up
```

The first startup downloads images, builds Go services, and initializes seeded MySQL and Keycloak volumes. Open **http://localhost:8080** for the GraphQL playground.

| Service | Address |
|---|---|
| Supergraph playground / GraphQL | http://localhost:8080 / http://localhost:8080/graphql |
| Accounts subgraph | http://localhost:8081/graphql |
| Catalog subgraph | http://localhost:8082/graphql |
| Reviews subgraph | http://localhost:8084/graphql |
| Keycloak | http://localhost:8085 |
| MySQL | localhost:3307, database `bookstore`, user/password `demo` / `demo` |
| Temporary login callback | http://localhost:8083/ (only while the login command runs) |

All published container ports bind to loopback. The sample users are **ada / ada** and **grace / grace**. Keycloak admin is **admin / admin**. These credentials are exclusively for this local demo.

```sh
curl -s http://localhost:8080/graphql \
  -H 'Content-Type: application/json' \
  --data '{"query":"{ books(first: 3) { title author { name } reviews(first: 5) { body rating author { name } } } }"}'
```

The seeded first book is `Computing Notes`, written by Ada Lovelace, with Grace Hopper's review `A pioneering read.`

## Sign in and write data

```sh
TOKEN=$(go run ./cmd/login)
```

The command prints a login URL to **stderr**. Open it in your browser and sign in at Keycloak as `ada` / `ada`. The callback validates state, issuer, the ID token, and nonce; the command then returns only the access token to stdout, which the shell captures in `TOKEN`. The token is short-lived (five minutes); run the command again when it expires. This command intentionally does not persist refresh tokens.

```sh
curl -s http://localhost:8080/graphql \
  -H 'Content-Type: application/json' \
  -H "Authorization: Bearer $TOKEN" \
  --data '{"query":"mutation { createReview(input: {bookId: \"1\", body: \"Great introduction\", rating: 5}) { id body createdAt author { name } book { title } } }"}'
```

For playground mutations, put `{"Authorization":"Bearer <token>"}` in its HTTP headers editor. Use `printf '%s\n' "$TOKEN"` locally if you need to copy it. Never commit tokens.

Reads and subscription events are intentionally public. Writes require a verified access token and a seeded bookstore profile. The server chooses the author from the verified `sub` claim: there is no caller-controlled author field. Invalid credentials fail at the HTTP boundary; missing credentials fail at the mutation resolver. Both direct subgraph access and router access enforce authorization.

## Subscriptions

In one playground tab, run:

```graphql
subscription {
  reviewCreated(bookId: "1") {
    id body rating
    author { name }
    book { title }
  }
}
```

Run `createReview` in another tab. The event goes through the router, which resolves the author and book fields across subgraphs. Clients use `ws://localhost:8080/graphql` with the `graphql-transport-ws` subprotocol. The wire sequence is `connection_init`, `connection_ack`, then `subscribe`; finish with `complete`.

The Reviews service publishes after a successful database insert. Its broker is bounded and process-local: it handles cancellation and terminates slow subscribers, but does not replay events or span replicas. Reconnect and query persisted reviews to recover missed data. Production replication needs shared pub/sub; transactional delivery guarantees need an outbox.

## Learn the concepts

Copy individual operations from [examples/queries.graphql](examples/queries.graphql).

| Concept | Example / location |
|---|---|
| Schema-first GraphQL | `internal/*/schema.graphqls` |
| Supergraph / subgraphs | `router/compose.yaml` and three services in `compose.yaml` |
| Federation entities | `@key`, `@external`, entity resolvers and `_entities` |
| Go equivalents of POJOs | Handwritten `store.User`; generated `catalog.Book`, `reviews.Review`, input structs |
| Field and root resolvers | `schema.resolvers.go` in each subgraph |
| Query / mutation / subscription | `book`, `createReview`, `reviewCreated` |
| Input validation and errors | `reviews/mutation.go`, `extensions.code` |
| Enums | `Genre`: `TECHNOLOGY`, `FICTION` |
| Custom scalar | `Time`, mapped to Go `time.Time`, serialized as RFC3339 |
| Interface | `Node`, implemented by `Book` and `ReadingList` |
| Union and inline fragments | `SearchResult = Book \| ReadingList` |
| Variables, aliases, fragments | `BookDetails` example |
| Directives | Federation directives and built-in `@include` / `@skip` |
| Nullability | Missing `book(id:)` returns `null`; required fields use `!` |
| Pagination | Bounded `first` / `offset` on books and reviews, deterministic ordering |
| Introspection | Enabled intentionally for exploration |
| OAuth/OIDC | `internal/auth`, `cmd/login`, imported Keycloak realm |

`ReadingList` is a derived search result grouping the matching books, not a stored entity. This is a focused example of common concepts, not an implementation of every Apollo product feature (such as GraphOS checks, client caching, or persisted query registries).

## Code layout and Go conventions

- `cmd/subgraph`: one binary, configured as Accounts, Catalog, or Reviews; graceful shutdown and HTTP timeouts.
- `cmd/login`: local Authorization Code + PKCE login helper.
- `internal/accounts`, `catalog`, `reviews`: generated contracts, typed models, and handwritten resolvers. Do not edit `generated.go`, `federation.go`, or `models_gen.go`.
- `internal/store`: explicit MySQL connection lifecycle and user access; domain queries use parameterized `database/sql` operations and request contexts.
- `internal/auth`: JWT signature verification using JWKS, plus issuer, audience, expiry, token type, and authorized-client checks.
- `internal/graphserver`: explicit transports, bounded query cache, request-body limit, complexity limit, and internal-error masking.
- `docker/mysql`: ordered initialization scripts and seed rows.
- `integration`: tests at the agreed GraphQL HTTP and WebSocket boundaries, using real MySQL and Keycloak.

The router uses a standard federation planner rather than custom query parsing/stitching. Composition runs locally using pinned `wgc`; all serving components are Go. The static router execution configuration is checked in, so ordinary startup does not require Node or schema generation.

The subgraphs share a MySQL database for simplicity. Tables have clear domain ownership, but this is not database-per-service isolation. Nested entity resolution can still cause multiple SQL reads: production workloads should add request-scoped batching/DataLoader and measure query plans. Pagination and complexity limits bound this example; they are not a substitute for workload-specific rate limits.

## Tests and development

```sh
make check              # vet + build/tests; integration tests explicitly skip without flags
make test-integration   # requires MySQL and Keycloak from make up
make test-e2e           # real stack, HTTP + WebSocket tests, Go race detector
make security           # scan Go code/dependencies against current Go advisories
```

Tests cover federation joins, entities, typed search, persistence, validation, anonymous writes, tampered signatures, ID/access-token separation, PKCE requirements, exact redirects, disabled password grants, and subscriptions both directly and through the router. Tests create additional reviews; they do not reset your database. OAuth tests act as a browser against Keycloak's login form; the application does not collect user passwords.

After editing schemas:

```sh
make generate
make compose
make up
make test-e2e
```

`make compose` runs pinned tooling in a Node container and updates `router/execution.json` using the schema files. `make up` rebuilds services. The router watches its execution configuration and reloads after composition. This disconnects active subscriptions; clients should reconnect. You can also force a reload with `docker compose restart router`.

To run one subgraph natively while its Compose counterpart is stopped:

```sh
docker compose stop accounts
LISTEN_ADDR=:8081 SUBGRAPH=accounts MYSQL_DSN='demo:demo@tcp(127.0.0.1:3307)/bookstore?parseTime=true' go run ./cmd/subgraph
```

The binary defaults to port 8080; set `LISTEN_ADDR=:8081` to avoid the router's port. Container routing still points at Compose services, so use the native endpoint directly while debugging, or change/recompose routing URLs deliberately.

```sh
make logs
make down               # preserves data
# Explicit destructive reset of this demo only:
docker compose down -v
make up
```

MySQL init scripts and Keycloak realm imports run only against fresh volumes. Editing those files does not migrate existing data. Use a migration tool and managed Keycloak configuration for a long-lived deployment.

## OAuth choices and local-only settings

Keycloak enables Authorization Code with **S256 PKCE**, exact redirect/logout URIs, a fixed API audience, five-minute access tokens, refresh-token rotation, brute-force protection, and disabled username editing/self-registration. Password, implicit, and service-account grants are disabled for the public client. User/profile linkage uses stable subject IDs. Tokens never go in query URLs or browser persistent storage. The CLI binds its callback to loopback, has a three-minute timeout, and validates state, issuer, nonce, and the ID token before releasing the access token.

The localhost example intentionally uses Keycloak `start-dev`, HTTP, a development database, known sample passwords, introspection, and public data. For deployment, use TLS, production Keycloak mode with an external supported database, managed secrets, stronger user enrollment/MFA policies, network-restricted subgraphs, rate limits, and an explicit authorization model for nonpublic data. Logout does not instantly revoke already-issued JWT access tokens; they expire after five minutes.

References: [OAuth 2.0 Security Best Current Practice (RFC 9700)](https://www.rfc-editor.org/rfc/rfc9700.html), [Keycloak containers](https://www.keycloak.org/server/containers), [gqlgen federation](https://gqlgen.com/recipes/federation/), and [Cosmo router](https://github.com/wundergraph/cosmo).
