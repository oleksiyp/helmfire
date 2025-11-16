.PHONY: build clean test lint install analysis coverage benchmarks docker release help

# Build variables
BINARY_NAME=helmfire
VERSION?=dev
GIT_COMMIT=$(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")
BUILD_DATE=$(shell date -u +"%Y-%m-%dT%H:%M:%SZ")
LDFLAGS=-ldflags "-X github.com/oleksiyp/helmfire/internal/version.Version=$(VERSION) \
                  -X github.com/oleksiyp/helmfire/internal/version.GitCommit=$(GIT_COMMIT) \
                  -X github.com/oleksiyp/helmfire/internal/version.BuildDate=$(BUILD_DATE)"

# Build the binary
build:
	@echo "Building $(BINARY_NAME) $(VERSION)..."
	go build $(LDFLAGS) -o $(BINARY_NAME) ./cmd/helmfire

# Build for multiple platforms
build-all:
	@echo "Building for multiple platforms..."
	@mkdir -p dist
	GOOS=linux GOARCH=amd64 go build $(LDFLAGS) -o dist/$(BINARY_NAME)-linux-amd64 ./cmd/helmfire
	GOOS=linux GOARCH=arm64 go build $(LDFLAGS) -o dist/$(BINARY_NAME)-linux-arm64 ./cmd/helmfire
	GOOS=darwin GOARCH=amd64 go build $(LDFLAGS) -o dist/$(BINARY_NAME)-darwin-amd64 ./cmd/helmfire
	GOOS=darwin GOARCH=arm64 go build $(LDFLAGS) -o dist/$(BINARY_NAME)-darwin-arm64 ./cmd/helmfire
	GOOS=windows GOARCH=amd64 go build $(LDFLAGS) -o dist/$(BINARY_NAME)-windows-amd64.exe ./cmd/helmfire
	@echo "Binaries built in dist/"

# Clean build artifacts
clean:
	@echo "Cleaning build artifacts..."
	rm -f $(BINARY_NAME)
	rm -rf dist/ build/ coverage.out coverage.html

# Run tests
test:
	@echo "Running tests..."
	go test -v -race -cover ./...

# Run tests with coverage
coverage:
	@echo "Running tests with coverage..."
	go test -v -race -coverprofile=coverage.out -covermode=atomic ./...
	go tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report generated: coverage.html"

# Run integration tests
test-integration:
	@echo "Running integration tests..."
	go test -v ./test/

# Run E2E tests
test-e2e:
	@echo "Running E2E tests..."
	go test -v -tags=e2e ./test/

# Run benchmarks
benchmarks:
	@echo "Running benchmarks..."
	go test -bench=. -benchmem ./test/ | tee benchmark.txt

# Run linters
lint:
	@echo "Running linters..."
	golangci-lint run --timeout 5m

# Format code
fmt:
	@echo "Formatting code..."
	gofmt -s -w .
	goimports -w .

# Install to GOPATH/bin
install:
	@echo "Installing $(BINARY_NAME)..."
	go install $(LDFLAGS) ./cmd/helmfire

# Download and analyze helm/helmfile sources
analysis:
	@echo "Downloading helm and helmfile sources for analysis..."
	mkdir -p analysis/sources
	cd analysis/sources && \
		git clone --depth 1 https://github.com/helmfile/helmfile.git && \
		git clone --depth 1 https://github.com/helm/helm.git
	@echo "Sources downloaded to analysis/sources/"
	@echo "See HELMFILE_ANALYSIS.md and HELM_PROJECT_ANALYSIS.md for details"

# Run in development mode
dev: build
	./$(BINARY_NAME) --help

# Build Docker image
docker:
	@echo "Building Docker image..."
	docker build \
		--build-arg VERSION=$(VERSION) \
		--build-arg GIT_COMMIT=$(GIT_COMMIT) \
		--build-arg BUILD_DATE=$(BUILD_DATE) \
		-t helmfire:$(VERSION) \
		-t helmfire:latest \
		.

# Run Docker image
docker-run:
	docker run --rm -it helmfire:latest

# Create release archives
release: build-all
	@echo "Creating release archives..."
	@mkdir -p dist
	cd dist && tar czf $(BINARY_NAME)-$(VERSION)-linux-amd64.tar.gz $(BINARY_NAME)-linux-amd64
	cd dist && tar czf $(BINARY_NAME)-$(VERSION)-linux-arm64.tar.gz $(BINARY_NAME)-linux-arm64
	cd dist && tar czf $(BINARY_NAME)-$(VERSION)-darwin-amd64.tar.gz $(BINARY_NAME)-darwin-amd64
	cd dist && tar czf $(BINARY_NAME)-$(VERSION)-darwin-arm64.tar.gz $(BINARY_NAME)-darwin-arm64
	cd dist && zip $(BINARY_NAME)-$(VERSION)-windows-amd64.zip $(BINARY_NAME)-windows-amd64.exe
	@echo "Release archives created in dist/"

# Generate checksums
checksums: release
	@echo "Generating checksums..."
	cd dist && sha256sum *.tar.gz *.zip > checksums.txt
	@echo "Checksums generated: dist/checksums.txt"

# Verify installation
verify:
	@echo "Verifying installation..."
	@which $(BINARY_NAME) > /dev/null || (echo "$(BINARY_NAME) not found in PATH" && exit 1)
	@$(BINARY_NAME) version
	@echo "Installation verified!"

# Help target
help:
	@echo "Helmfire Makefile Targets:"
	@echo ""
	@echo "Build:"
	@echo "  build          - Build binary for current platform"
	@echo "  build-all      - Build for all platforms"
	@echo "  install        - Install to GOPATH/bin"
	@echo "  clean          - Remove build artifacts"
	@echo ""
	@echo "Testing:"
	@echo "  test           - Run unit tests"
	@echo "  test-integration - Run integration tests"
	@echo "  test-e2e       - Run end-to-end tests"
	@echo "  coverage       - Generate coverage report"
	@echo "  benchmarks     - Run performance benchmarks"
	@echo ""
	@echo "Quality:"
	@echo "  lint           - Run linters"
	@echo "  fmt            - Format code"
	@echo ""
	@echo "Docker:"
	@echo "  docker         - Build Docker image"
	@echo "  docker-run     - Run Docker image"
	@echo ""
	@echo "Release:"
	@echo "  release        - Create release archives"
	@echo "  checksums      - Generate checksums"
	@echo ""
	@echo "Other:"
	@echo "  dev            - Build and show help"
	@echo "  verify         - Verify installation"
	@echo "  analysis       - Download helm/helmfile sources"
	@echo "  help           - Show this help"
