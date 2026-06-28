# Build stage - Go binary compilation
FROM golang:1.25-alpine AS builder
WORKDIR /app

# Cache dependencies layer
COPY workers/deploy/go.mod workers/deploy/go.sum ./
RUN go mod download

# Build binary
COPY workers/deploy/*.go ./
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build \
    -ldflags="-s -w -X main.BuildTime=$(date -u '+%Y-%m-%d_%H:%M:%S')" \
    -trimpath \
    -o /nidus-deploy-worker .

# Runtime stage - Minimal image with only runtime dependencies
FROM alpine:3.20

# Install only essential tools
RUN apk add --no-cache \
    ca-certificates \
    git \
    git-lfs \
    docker-cli \
    openssh-client \
    && rm -rf /var/cache/apk/*

# Security: run as non-root
RUN addgroup -S nidus && adduser -S nidus -G nidus

WORKDIR /app

# Copy binary from builder
COPY --from=builder /nidus-deploy-worker .
RUN chmod +x ./nidus-deploy-worker

# Environment variables
ENV NIDUS_DEPLOYS_DIR=/data/deploys \
    BUILDKIT_ENABLED=true \
    MAX_CONCURRENT_BUILDS=50 \
    DOCKER_TIMEOUT_SECONDS=120

# Create deploy directory
RUN mkdir -p /data/deploys && chown -R nidus:nidus /data

# Health check
HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
    CMD wget --quiet --tries=1 --spider http://localhost:8081/health || exit 1

VOLUME /data

# Run as non-root
USER nidus

# Expose metrics and health check port
EXPOSE 8081

CMD ["./nidus-deploy-worker"]
