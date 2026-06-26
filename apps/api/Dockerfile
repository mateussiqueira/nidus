FROM golang:1.25-alpine AS builder
WORKDIR /app
COPY apps/api/go.mod apps/api/go.sum ./
RUN go mod download
COPY apps/api/*.go ./
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /nidus-api .

FROM alpine:3.20
RUN apk add --no-cache ca-certificates
COPY --from=builder /nidus-api /usr/local/bin/nidus-api
EXPOSE 3001
CMD ["nidus-api"]
