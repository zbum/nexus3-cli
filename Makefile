BINARY      := nexus3-cli
PKG         := github.com/zbum/nexus3-cli
DIST        := dist
VERSION     ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo dev)
LDFLAGS     := -s -w -X $(PKG)/internal/cli.Version=$(VERSION)
GO          ?= go

.PHONY: all build clean test lint fmt run build-all help

all: build

help:
	@echo "Targets: build clean test lint fmt run build-all"

build:
	@mkdir -p $(DIST)
	$(GO) build -trimpath -ldflags '$(LDFLAGS)' -o $(DIST)/$(BINARY) .

clean:
	rm -rf $(DIST)

test:
	$(GO) test ./...

lint:
	$(GO) vet ./...
	@command -v golangci-lint >/dev/null && golangci-lint run ./... || echo "golangci-lint not installed, skipping"

fmt:
	$(GO) fmt ./...

run: build
	./$(DIST)/$(BINARY)

# Cross-compile release binaries into dist/
PLATFORMS := \
	linux/amd64 \
	linux/arm64 \
	darwin/amd64 \
	darwin/arm64 \
	windows/amd64

build-all:
	@mkdir -p $(DIST)
	@for p in $(PLATFORMS); do \
		os=$${p%/*}; arch=$${p#*/}; \
		ext=""; [ "$$os" = "windows" ] && ext=".exe"; \
		out=$(DIST)/$(BINARY)-$$os-$$arch$$ext; \
		echo "→ $$out"; \
		GOOS=$$os GOARCH=$$arch CGO_ENABLED=0 $(GO) build -trimpath -ldflags '$(LDFLAGS)' -o $$out . || exit 1; \
	done