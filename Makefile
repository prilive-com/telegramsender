.PHONY: all clean build test lint docker-build docker-run

# Variables
GO = go
DOCKER = docker
COMPOSE = docker-compose
GOFMT = gofmt
GOVET = $(GO) vet
GOLINT = golangci-lint
MAIN_PKG = ./example
APP_NAME = telegramsender

# Targets
all: clean lint security test build

clean:
	rm -rf ./bin

build:
	mkdir -p bin
	$(GO) build -o bin/$(APP_NAME) $(MAIN_PKG)

test:
	$(GO) test -v ./...

security:
	@if command -v gosec > /dev/null; then \
		echo "Running gosec security scanner..."; \
		gosec -quiet ./...; \
	else \
		echo "gosec not installed. Install with: go install github.com/securego/gosec/v2/cmd/gosec@latest"; \
		exit 1; \
	fi

lint:
	$(GOVET) ./...
	@echo "Go vet passed"

fmt:
	$(GOFMT) -s -w .

docker-build:
	$(COMPOSE) build

docker-run:
	$(COMPOSE) up -d

docker-logs:
	$(COMPOSE) logs -f

# Default target
default: build