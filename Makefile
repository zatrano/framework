# ZATRANO — optional Makefile for POSIX shells (Git Bash, macOS, Linux).
# On Windows PowerShell, use: go run ./cmd/zatrano serve

.PHONY: build test fmt vet lint run doctor air gen-example openapi-export verify verify-race config-validate
APP := ./cmd/zatrano

build:
	go build -o bin/zatrano $(APP)

test:
	go test ./... -count=1

fmt:
	go fmt ./...

vet:
	go vet ./...

lint:
	golangci-lint run

run:
	go run $(APP) serve

doctor:
	go run $(APP) doctor

air:
	air

gen-example:
	go run $(APP) gen module example_item --dry-run

openapi-export:
	go run $(APP) openapi export --output api/openapi.merged.yaml

# Offline config check (no DB/Redis); same flags as CLI
config-validate:
	go run $(APP) config validate

# Same checks as: go run ./cmd/zatrano verify
verify:
	go run $(APP) verify

verify-race:
	go run $(APP) verify --race
