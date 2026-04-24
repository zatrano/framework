# ─────────────────────────────────────────────────────────────────────────────
#  ZATRANO Enterprise — Makefile
# ─────────────────────────────────────────────────────────────────────────────

APP_NAME   := zatrano
BUILD_DIR  := ./bin
MAIN       := ./main.go
DB_CMD     := ./database/cmd/main.go
VERSION    ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
GOFLAGS    := CGO_ENABLED=0 GOOS=linux GOARCH=amd64
LDFLAGS    := -ldflags="-s -w -extldflags '-static' -X main.version=$(VERSION)"

.PHONY: help run dev build clean migrate seed tidy lint test test-unit test-integration test-race docker-build

## help: Yardım mesajını göster
help:
	@grep -E '^## ' Makefile | sed 's/## //'

## tidy: Bağımlılıkları güncelle ve doğrula
tidy:
	go mod tidy
	go mod verify

## dev: Geliştirme modunda çalıştır
dev:
	APP_ENV=development go run $(MAIN)

## build: Production binary derle
build:
	mkdir -p $(BUILD_DIR)
	$(GOFLAGS) go build $(LDFLAGS) -trimpath -o $(BUILD_DIR)/$(APP_NAME) $(MAIN)
	@echo "✓ Build: $(BUILD_DIR)/$(APP_NAME) (version=$(VERSION))"

## run: Build + çalıştır
run: build
	$(BUILD_DIR)/$(APP_NAME)

## migrate: DB migration çalıştır
migrate:
	go run $(DB_CMD) -migrate

## seed: DB seed çalıştır
seed:
	go run $(DB_CMD) -seed

## migrate-seed: Migration + Seed
migrate-seed:
	go run $(DB_CMD) -migrate -seed

## clean: Build artefaktlarını temizle
clean:
	rm -rf $(BUILD_DIR) coverage.out coverage.html

## test-unit: Unit testleri çalıştır (DB/Redis gerektirmez)
test-unit:
	go test -v -race -count=1 ./tests/unit/...

## test-integration: Integration testleri docker ile çalıştır
test-integration:
	docker-compose -f docker-compose.test.yml up --abort-on-container-exit --build
	docker-compose -f docker-compose.test.yml down -v

## test: Unit + coverage raporu
test:
	go test -race -coverprofile=coverage.out -covermode=atomic ./tests/unit/...
	go tool cover -func=coverage.out | tail -1
	go tool cover -html=coverage.out -o coverage.html
	@echo "✓ Coverage raporu: coverage.html"

## test-race: Race condition kontrolü
test-race:
	go test -race -count=3 ./tests/unit/...

## lint: golangci-lint çalıştır
lint:
	golangci-lint run --timeout=5m ./...

## docker-build: Docker image oluştur
docker-build:
	docker build \
		--build-arg VERSION=$(VERSION) \
		--target runner \
		-t $(APP_NAME):$(VERSION) \
		-t $(APP_NAME):latest \
		.
	@echo "✓ Docker image: $(APP_NAME):$(VERSION)"

## vet: Go vet çalıştır
vet:
	go vet ./...
