BINARY := cyberpilot
PACKAGE := ./cmd/cyberpilot
VERSION ?= dev
COMMIT ?= $(shell git rev-parse --short HEAD 2>/dev/null || printf unknown)
BUILD_DATE ?= $(shell date -u +%Y-%m-%dT%H:%M:%SZ)
LDFLAGS := -s -w \
	-X github.com/cyberkit-x/cyberpilot/internal/buildinfo.version=$(VERSION) \
	-X github.com/cyberkit-x/cyberpilot/internal/buildinfo.commit=$(COMMIT) \
	-X github.com/cyberkit-x/cyberpilot/internal/buildinfo.date=$(BUILD_DATE)

.PHONY: build test check

build:
	CGO_ENABLED=0 go build -trimpath -ldflags "$(LDFLAGS)" -o bin/$(BINARY) $(PACKAGE)

test:
	go test ./...

check:
	test -z "$$(gofmt -l .)" || { echo "Go files need formatting"; gofmt -l .; exit 1; }
	go vet ./...
	go test ./...
