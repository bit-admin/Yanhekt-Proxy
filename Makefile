.PHONY: build build-linux build-linux-arm run clean docker-build docker-run test fmt lint

# Binary name
BINARY=video-proxy

# Build the binary
build:
	go build -o $(BINARY) ./cmd/proxy

# Build for Linux (for deployment)
build-linux:
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags="-w -s" -o $(BINARY)-linux ./cmd/proxy

# Build for Linux ARM64 (for ARM servers like Raspberry Pi, AWS Graviton)
build-linux-arm:
	CGO_ENABLED=0 GOOS=linux GOARCH=arm64 go build -ldflags="-w -s" -o $(BINARY)-linux-arm64 ./cmd/proxy

# Run the server locally
run: build
	./$(BINARY)

# Run with custom config
run-dev:
	PORT=8080 LOG_LEVEL=debug go run ./cmd/proxy

# Clean build artifacts
clean:
	rm -f $(BINARY) $(BINARY)-linux $(BINARY)-linux-arm64
	go clean

# Build Docker image
docker-build:
	docker build -t video-proxy:latest .

# Run Docker container
docker-run:
	docker run -p 8080:8080 -v $(PWD)/mappings.json:/app/mappings.json video-proxy:latest

# Run tests
test:
	go test -v ./...

# Format code
fmt:
	go fmt ./...

# Run linter
lint:
	golangci-lint run

# Download dependencies
deps:
	go mod download
	go mod tidy

# Show help
help:
	@echo "Available targets:"
	@echo "  build        - Build the binary"
	@echo "  build-linux  - Build for Linux amd64 deployment"
	@echo "  build-linux-arm - Build for Linux arm64 deployment"
	@echo "  run          - Build and run locally"
	@echo "  run-dev      - Run in development mode"
	@echo "  clean        - Clean build artifacts"
	@echo "  docker-build - Build Docker image"
	@echo "  docker-run   - Run Docker container"
	@echo "  test         - Run tests"
	@echo "  fmt          - Format code"
	@echo "  lint         - Run linter"
	@echo "  deps         - Download dependencies"
