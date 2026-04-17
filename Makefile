export PATH := $(HOME)/bin/go/bin:$(PATH)

BINARY  := filesystem-gateway
VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
LDFLAGS := -ldflags "-s -w -X main.version=$(VERSION)"
ENV_FILE := $(dir $(abspath $(lastword $(MAKEFILE_LIST))))/../.env

GOLANGCI_LINT_VERSION := $(shell cat .golangci-version)
GOLANGCI_LINT := ./bin/golangci-lint

.PHONY: build test cover lint run clean hooks deploy

build:
	@mkdir -p bin
	go build -trimpath $(LDFLAGS) -o bin/$(BINARY) ./cmd/server

test:
	go test -race ./...

cover:
	go test -race -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out

lint: $(GOLANGCI_LINT)
	$(GOLANGCI_LINT) run ./...

$(GOLANGCI_LINT): .golangci-version
	@mkdir -p ./bin
	@echo "Installing golangci-lint $(GOLANGCI_LINT_VERSION)..."
	@curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/HEAD/install.sh | sh -s -- -b ./bin $(GOLANGCI_LINT_VERSION)

run:
	go run ./cmd/server

hooks:
	git config core.hooksPath .githooks

clean:
	rm -rf bin/ coverage.out

deploy:
	@set -a && . $(ENV_FILE) && set +a && \
	echo "==> Restarting $(BINARY) in Nomad..." && \
	nomad job restart -reschedule -yes $(BINARY) && \
	echo "==> Done! $(BINARY) is redeploying."
