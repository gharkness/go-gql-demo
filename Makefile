.PHONY: up down logs generate compose test test-integration test-e2e check login security
up:
	docker compose up -d --build --wait --wait-timeout 180
	@echo 'GraphQL playground: http://localhost:8080'
	@echo 'Keycloak: http://localhost:8085'
down:
	docker compose down
logs:
	docker compose logs -f
# Generated Go code and router execution.json are checked in; generation is needed only after schema changes.
generate:
	go tool gqlgen generate --config internal/accounts/gqlgen.yml
	go tool gqlgen generate --config internal/catalog/gqlgen.yml
	go tool gqlgen generate --config internal/reviews/gqlgen.yml
	gofmt -w cmd internal integration
compose:
	docker run --rm -v "$(CURDIR):/work" -w /work/router node:22-alpine sh -c 'npm ci --no-audit --no-fund && npm run compose'
# Without environment flags, service-dependent tests skip explicitly.
test:
	go test ./...
test-integration:
	INTEGRATION=1 go test -count=1 ./integration
test-e2e:
	INTEGRATION=1 E2E=1 go test -race -count=1 -v ./integration
check:
	go vet ./...
	go test ./...
login:
	go run ./cmd/login

security:
	go run golang.org/x/vuln/cmd/govulncheck@latest ./...
