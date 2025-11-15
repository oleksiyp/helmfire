.PHONY: build clean test lint install analysis

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
	go build $(LDFLAGS) -o $(BINARY_NAME) ./cmd/helmfire

# Clean build artifacts
clean:
	rm -f $(BINARY_NAME)
	rm -rf dist/ build/

# Run tests
test:
	go test -v -race -cover ./...

# Run linters
lint:
	golangci-lint run

# Install to GOPATH/bin
install:
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
