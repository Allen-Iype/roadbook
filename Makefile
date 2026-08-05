# Roadbook build targets.

.PHONY: generate build test vet

# Regenerate everything derived from api/openapi.yaml — the single contract.
# Generated files are committed and never edited by hand.
generate:
	go tool oapi-codegen -config api/codegen.yaml api/openapi.yaml
	cd web && npm run generate

build:
	go build -o bin/roadbook ./cmd/roadbook

# Scoped to our packages: web/node_modules ships a stray Go package that
# ./... would otherwise pick up. The script resolves a test database (env →
# local Postgres → Docker compose → visible skip) so the store's DB-backed
# tests run by default; skipping is the exception.
test:
	sh scripts/test.sh

vet:
	go vet ./cmd/... ./internal/... ./migrations/...
	gofmt -l cmd internal migrations
