# Makefile for the another-me project

# Go parameters
GOBASE := $(shell pwd)
PKG_LIST := $(shell go list ./... | grep -v /vendor/)

# Docker parameters
DOCKER_COMPOSE_FILE := docker-compose.yaml

.PHONY: all actions test test-unit test-integration lint tidy docker-up docker-down docker-clean-restart docker-logs help

all: test lint tidy

# ===================================================================================
# GitHub Actions (local)
# ===================================================================================
actions:
	@echo "Installing/Updating act..."
	@go install github.com/nektos/act@latest
	@echo "Running act..."
	@act

# ===================================================================================
# Testing
# ===================================================================================
test: test-unit

test-unit:
	@echo "Running unit tests..."
	@go test -v -race -coverprofile=coverage-unit.out $(PKG_LIST)

test-integration:
	@echo "Running integration tests..."
	@echo "Ensure Docker services (PostgreSQL, etc.) are running before executing integration tests."
	@go test -v -race -count=1 -tags=integration $(PKG_LIST)

test-all: test-unit test-integration
	@echo "All tests completed."

# ===================================================================================
# Go tools
# ===================================================================================
lint:
	@echo "Running linters..."
	@golangci-lint-v2 run ./... || golangci-lint run ./...

tidy:
	@echo "Tidying go modules..."
	@go mod tidy

# ===================================================================================
# Docker
# ===================================================================================
docker-up:
	@echo "Starting Docker services in detached mode..."
	@docker compose -f $(DOCKER_COMPOSE_FILE) up -d

docker-down:
	@echo "Stopping Docker services..."
	@docker compose -f $(DOCKER_COMPOSE_FILE) down

docker-clean-restart:
	@echo "Stopping Docker services..."
	@docker compose -f $(DOCKER_COMPOSE_FILE) down
	@rm -rf $(GOBASE)/.docker-data
	@echo "Starting Docker services in detached mode..."
	@docker compose -f $(DOCKER_COMPOSE_FILE) up -d

docker-logs:
	@echo "Showing Docker service logs..."
	@docker compose -f $(DOCKER_COMPOSE_FILE) logs -f

# ===================================================================================
# Help
# ===================================================================================
help:
	@echo "-------------------------------------------------------------------------"
	@echo "Usage: make <target>"
	@echo ""
	@echo "Available targets are:"
	@echo "  all                  Runs test, lint, and tidy."
	@echo "  actions              Runs GitHub Actions locally."
	@echo "  test                 Alias for test-unit."
	@echo "  test-unit            Runs all unit tests."
	@echo "  test-integration     Runs all integration tests (requires Docker services to be up)."
	@echo "  test-all             Runs all unit and integration tests."
	@echo "  lint                 Runs golangci-lint."
	@echo "  tidy                 Runs go mod tidy."
	@echo "  docker-up            Starts Docker services (PostgreSQL, Neo4j, Qdrant) in detached mode."
	@echo "  docker-down          Stops Docker services."
	@echo "  docker-clean-restart Stops Docker services and removes the .docker-data directory, then starts Docker services in detached mode."
	@echo "  docker-logs          Shows logs for running Docker services."
	@echo "  help                 Shows this help message."
	@echo "-------------------------------------------------------------------------"

# Default target
.DEFAULT_GOAL := help 