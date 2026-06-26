FROM rust:1.85-alpine AS builder
RUN apk add --no-cache musl-dev
WORKDIR /app
COPY apps/proxy/Cargo.toml apps/proxy/Cargo.lock ./
RUN mkdir src && echo "fn main() {}" > src/main.rs && cargo build --release 2>/dev/null || true
COPY apps/proxy/src ./src
RUN touch src/main.rs && cargo build --release

FROM alpine:3.20
RUN apk add --no-cache ca-certificates
COPY --from=builder /app/target/release/nidus-proxy /usr/local/bin/nidus-proxy
EXPOSE 3080
CMD ["nidus-proxy"]
