# Video Storage AI - API Makefile
# Go parameters
GOCMD=go
GOBUILD=$(GOCMD) build
GOCLEAN=$(GOCMD) clean
GOTEST=$(GOCMD) test
GOGET=$(GOCMD) get
GOMOD=$(GOCMD) mod
BINARY_NAME=video-storage-ai
BINARY_WINDOWS=$(BINARY_NAME).exe
MAIN_PATH=./cmd/server

# Build flags
LDFLAGS=-ldflags "-s -w"

.PHONY: all build clean test coverage deps run dev help install

# Default target
all: deps build

## help: Display this help message
help:
	@echo "Video Storage AI - API Build Commands"
	@echo ""
	@echo "Usage: make [target]"
	@echo ""
	@echo "Targets:"
	@grep -E '^## ' $(MAKEFILE_LIST) | sed 's/## /  /'

## deps: Download and install dependencies
deps:
	@echo "Downloading dependencies..."
	$(GOMOD) download
	$(GOMOD) tidy

## build: Build the application (optimized)
build: deps
	@echo "Building $(BINARY_NAME)..."
	$(GOBUILD) $(LDFLAGS) -o bin/$(BINARY_WINDOWS) $(MAIN_PATH)
	@echo "Build complete: bin/$(BINARY_WINDOWS)"

## build-debug: Build the application with debug symbols
build-debug: deps
	@echo "Building $(BINARY_NAME) with debug symbols..."
	$(GOBUILD) -o bin/$(BINARY_WINDOWS) $(MAIN_PATH)
	@echo "Debug build complete: bin/$(BINARY_WINDOWS)"

## run: Run the application without building binary
run: deps
	@echo "Running application..."
	$(GOCMD) run $(MAIN_PATH)/main.go

## dev: Run the application in development mode with auto-reload (requires air)
dev:
	@echo "Starting development server with air..."
	@if command -v air > /dev/null; then \
		air; \
	else \
		echo "Error: 'air' is not installed. Install it with: go install github.com/cosmtrek/air@latest"; \
		echo "Falling back to 'go run'..."; \
		$(MAKE) run; \
	fi

## test: Run tests
test:
	@echo "Running tests..."
	$(GOTEST) -v ./...

## test-coverage: Run tests with coverage
test-coverage:
	@echo "Running tests with coverage..."
	$(GOTEST) -v -coverprofile=coverage.out ./...
	$(GOCMD) tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report generated: coverage.html"

## clean: Remove build artifacts and cache
clean:
	@echo "Cleaning..."
	$(GOCLEAN)
	rm -rf bin/
	rm -rf data/
	rm -f coverage.out coverage.html
	@echo "Clean complete"

## install: Install development tools
install:
	@echo "Installing development tools..."
	$(GOCMD) install github.com/cosmtrek/air@latest
	@echo "Tools installed successfully"

## db-reset: Reset the database (WARNING: deletes all data)
db-reset:
	@echo "Resetting database..."
	rm -rf data/
	@echo "Database reset complete. It will be recreated on next run."

## lint: Run linter (requires golangci-lint)
lint:
	@echo "Running linter..."
	@if command -v golangci-lint > /dev/null; then \
		golangci-lint run; \
	else \
		echo "Error: 'golangci-lint' is not installed"; \
		echo "Install it from: https://golangci-lint.run/usage/install/"; \
	fi

## fmt: Format code
fmt:
	@echo "Formatting code..."
	$(GOCMD) fmt ./...

## vet: Run go vet
vet:
	@echo "Running go vet..."
	$(GOCMD) vet ./...

## mod-update: Update dependencies
mod-update:
	@echo "Updating dependencies..."
	$(GOMOD) tidy
	$(GOMOD) download

## check: Run fmt, vet, and test
check: fmt vet test
	@echo "All checks passed!"
