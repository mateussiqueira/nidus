# Build stage
FROM golang:1.23-alpine AS builder
WORKDIR /app
COPY workers/deploy/go.mod workers/deploy/go.sum ./
RUN go mod download
COPY workers/deploy/*.go ./
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /nidus-deploy-worker .

# Production stage
FROM alpine:3.19
RUN apk add --no-cache ca-certificates git docker-cli
WORKDIR /app
COPY --from=builder /nidus-deploy-worker .
ENV NIDUS_DEPLOYS_DIR=/data/deploys
VOLUME /data
CMD ["./nidus-deploy-worker"]
