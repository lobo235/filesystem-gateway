export PATH := $(HOME)/bin/go/bin:$(PATH)

BINARY  := filesystem-gateway
VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
LDFLAGS := -ldflags "-s -w -X main.version=$(VERSION)"
ENV_FILE := $(dir $(abspath $(lastword $(MAKEFILE_LIST))))/../.env

.PHONY: build test cover lint run clean hooks deploy

build:
	@mkdir -p bin
	go build -trimpath $(LDFLAGS) -o bin/$(BINARY) ./cmd/server

test:
	go test ./...

cover:
	go test -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out

lint:
	golangci-lint run ./...

run:
	go run ./cmd/server

hooks:
	git config core.hooksPath .githooks

clean:
	rm -f $(BINARY) coverage.out

deploy:
	@set -a && . $(ENV_FILE) && set +a && \
	echo "==> Restarting $(BINARY) in Nomad..." && \
	nomad job restart -reschedule -yes $(BINARY) && \
	echo "==> Done! $(BINARY) is redeploying."
