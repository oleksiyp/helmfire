# Build stage
FROM golang:1.22-alpine AS builder

# Install build dependencies
RUN apk add --no-cache git make

# Set working directory
WORKDIR /build

# Copy go mod files
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Build arguments for version info
ARG VERSION=dev
ARG GIT_COMMIT=unknown
ARG BUILD_DATE=unknown

# Build binary
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build \
    -ldflags "-X github.com/oleksiyp/helmfire/internal/version.Version=${VERSION} \
              -X github.com/oleksiyp/helmfire/internal/version.GitCommit=${GIT_COMMIT} \
              -X github.com/oleksiyp/helmfire/internal/version.BuildDate=${BUILD_DATE} \
              -w -s" \
    -o helmfire ./cmd/helmfire

# Runtime stage
FROM alpine:3.18

# Install runtime dependencies
RUN apk add --no-cache \
    ca-certificates \
    bash \
    curl \
    git

# Install helm
ARG HELM_VERSION=3.13.3
RUN curl -fsSL https://get.helm.sh/helm-v${HELM_VERSION}-linux-amd64.tar.gz | tar xz && \
    mv linux-amd64/helm /usr/local/bin/helm && \
    rm -rf linux-amd64

# Install kubectl
ARG KUBECTL_VERSION=1.28.4
RUN curl -fsSLO "https://dl.k8s.io/release/v${KUBECTL_VERSION}/bin/linux/amd64/kubectl" && \
    chmod +x kubectl && \
    mv kubectl /usr/local/bin/kubectl

# Copy helmfire binary from builder
COPY --from=builder /build/helmfire /usr/local/bin/helmfire

# Create non-root user
RUN addgroup -g 1000 helmfire && \
    adduser -D -u 1000 -G helmfire helmfire && \
    mkdir -p /home/helmfire/.helmfire && \
    chown -R helmfire:helmfire /home/helmfire

# Switch to non-root user
USER helmfire
WORKDIR /workspace

# Set entrypoint
ENTRYPOINT ["/usr/local/bin/helmfire"]

# Default command
CMD ["--help"]

# Labels
LABEL org.opencontainers.image.title="Helmfire"
LABEL org.opencontainers.image.description="Dynamic Kubernetes deployment tool with live chart/image substitution"
LABEL org.opencontainers.image.url="https://github.com/oleksiyp/helmfire"
LABEL org.opencontainers.image.source="https://github.com/oleksiyp/helmfire"
LABEL org.opencontainers.image.vendor="Helmfire"
LABEL org.opencontainers.image.licenses="Apache-2.0"
