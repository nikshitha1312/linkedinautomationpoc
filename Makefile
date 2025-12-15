# LinkedIn Automation PoC - Makefile
# For Windows, use 'make' with MinGW or use the PowerShell scripts

.PHONY: all build run test clean deps

# Default target
all: deps build

# Install dependencies
deps:
	go mod download
	go mod tidy

# Build the application
build:
	go build -o bin/linkedin-automation.exe ./cmd/main.go

# Run in interactive mode
run:
	go run ./cmd/main.go -mode=interactive

# Run search mode
search:
	go run ./cmd/main.go -mode=search -search="Software Engineer"

# Run with verbose logging
run-verbose:
	go run ./cmd/main.go -mode=interactive -verbose

# Run tests
test:
	go test -v ./...

# Run tests with coverage
test-coverage:
	go test -cover -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html

# Clean build artifacts
clean:
	rm -rf bin/
	rm -rf data/
	rm -rf logs/
	rm -f coverage.out coverage.html

# Format code
fmt:
	go fmt ./...

# Lint code
lint:
	golangci-lint run

# Generate documentation
docs:
	godoc -http=:6060

# Create necessary directories
init:
	mkdir -p data logs bin
