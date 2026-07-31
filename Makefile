# Roadbook build targets.

.PHONY: generate build test vet

# Regenerate everything derived from api/openapi.yaml — the single contract.
# Generated files are committed and never edited by hand.
generate:
	go tool oapi-codegen -config api/codegen.yaml api/openapi.yaml
	cd web && npm run generate

build:
	go build -o bin/roadbook ./cmd/roadbook

test:
	go test ./...

vet:
	go vet ./...
	gofmt -l .
