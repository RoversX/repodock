APP := repodock

.PHONY: run build fmt tidy dist

VERSION ?= dev
COMMIT ?= $(shell git rev-parse --short HEAD 2>/dev/null || echo none)
DATE ?= $(shell date -u +"%Y-%m-%dT%H:%M:%SZ")

run:
	go run ./cmd/$(APP)

build:
	go build -ldflags "-X github.com/roversx/repodock/internal/buildinfo.Version=$(VERSION) -X github.com/roversx/repodock/internal/buildinfo.Commit=$(COMMIT) -X github.com/roversx/repodock/internal/buildinfo.Date=$(DATE)" -o bin/$(APP) ./cmd/$(APP)

dist:
	VERSION=$(VERSION) COMMIT=$(COMMIT) DATE=$(DATE) scripts/build-release.sh

fmt:
	go fmt ./...

tidy:
	go mod tidy
