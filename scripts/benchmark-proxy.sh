#!/bin/bash
# Nidus Proxy Benchmark — Rust vs Go vs Caddy vs Nginx
set -e

TARGET="${1:-http://127.0.0.1:3000}"  # dashboard as upstream
CONCURRENCY="${2:-100}"
REQUESTS="${3:-10000}"

echo "═══════════════════════════════════════════"
echo "  Nidus Proxy Benchmark Suite"
echo "  Target: $TARGET"
echo "  Concurrency: $CONCURRENCY"
echo "  Requests: $REQUESTS"
echo "═══════════════════════════════════════════"

# Install wrk if needed
which wrk || apt-get install -y -qq wrk

# ── Rust Proxy ──
echo ""
echo "🦀 Rust Proxy (nidus-proxy v0.2.0)"
# Kill old instances
pkill -f nidus-proxy 2>/dev/null || true
# Start Rust proxy on port 8089
/root/nidus/rust/target/release/nidus-proxy &
sleep 2
# Redirect it to dashboard
RUST_PID=$!
echo "  PID: $RUST_PID, Memory: $(ps -o rss= -p $RUST_PID | tr -d ' ') KB"
wrk -t4 -c$CONCURRENCY -d10s --latency http://127.0.0.1:8089/dashboard 2>&1 | tee /tmp/rust-bench.txt
kill $RUST_PID 2>/dev/null || true
sleep 1

# ── Go Proxy ──
echo ""
echo "🐹 Go Proxy (nidus-proxy)"
pkill -f "nidus-proxy" 2>/dev/null || true
/root/nidus/apps/proxy/nidus-proxy &
GO_PID=$!
sleep 2
echo "  PID: $GO_PID, Memory: $(ps -o rss= -p $GO_PID | tr -d ' ') KB"
wrk -t4 -c$CONCURRENCY -d10s --latency http://127.0.0.1:8080/dashboard 2>&1 | tee /tmp/go-bench.txt
kill $GO_PID 2>/dev/null || true
sleep 1

# ── Caddy ──
echo ""
echo "🐹 Caddy (reverse proxy)"
wrk -t4 -c$CONCURRENCY -d10s --latency https://nidus.app 2>&1 | tee /tmp/caddy-bench.txt

# ── Summary ──
echo ""
echo "═══════════════════════════════════════════"
echo "  RESULTS SUMMARY"
echo "═══════════════════════════════════════════"
for f in /tmp/rust-bench.txt /tmp/go-bench.txt /tmp/caddy-bench.txt; do
    name=$(grep -l "$f" /dev/null || basename $f)
    reqs=$(grep "Requests/sec" $f | awk '{print $2}')
    avg=$(grep "Average" $f | head -1 | awk '{print $2}')
    echo "$name: ${reqs:-N/A} req/s, avg latency: ${avg:-N/A}"
done
