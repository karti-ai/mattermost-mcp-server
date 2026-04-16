# Mattermost MCP Server Makefile

.PHONY: build test clean dev lint

BINARY_NAME=mattermost-mcp-server
VERSION?=$(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
BUILD_TIME=$(shell date -u '+%Y-%m-%d_%H:%M:%S')
LDFLAGS=-ldflags "-X main.Version=$(VERSION)"

build:
	go build $(LDFLAGS) -o $(BINARY_NAME) .

test:
	go test -v ./...

clean:
	rm -f $(BINARY_NAME)
	go clean

dev:
	go run . -log-level=debug

lint:
	golangci-lint run

mod:
	go mod tidy
	go mod verify

.DEFAULT_GOAL := build
