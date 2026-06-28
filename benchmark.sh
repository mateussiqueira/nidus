#!/bin/bash
# Nidus Performance Benchmark Suite
# Run: ./benchmark.sh

set -e

echo "========================================="
echo "  Nidus Performance Benchmark Suite"
echo "========================================="
echo ""

API_URL="${API_URL:-http://localhost:3001}"
RESULTS_FILE="benchmark-results-$(date +%Y%m%d-%H%M%S).json"

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m'

# Detect OS for nanosecond timing
if command -v $DATE_CMD &>/dev/null; then
    DATE_CMD="$DATE_CMD"
else
    DATE_CMD="date"
fi

# Benchmark functions
measure_time() {
    local start=$($DATE_CMD +%s%N 2>/dev/null || $DATE_CMD +%s000000000)
    eval "$2" > /dev/null 2>&1
    local end=$($DATE_CMD +%s%N 2>/dev/null || $DATE_CMD +%s000000000)
    local duration=$(( (end - start) / 1000000 ))
    echo "$duration"
}

measure_throughput() {
    local requests=$1
    local url=$2
    local concurrent=$3
    
    local start=$($DATE_CMD +%s%N)
    for i in $(seq 1 $requests); do
        curl -s "$url" > /dev/null &
        if (( i % concurrent == 0 )); then
            wait
        fi
    done
    wait
    local end=$($DATE_CMD +%s%N)
    local duration=$(( (end - start) / 1000000 ))
    local rps=$(( requests * 1000 / duration ))
    echo "$rps"
}

echo -e "${YELLOW}[1/6] API Response Time${NC}"
api_time=$(measure_time "curl -s $API_URL/health")
echo -e "  Health endpoint: ${GREEN}${api_time}ms${NC}"

echo ""
echo -e "${YELLOW}[2/6] Database Connection Pool${NC}"
db_start=$($DATE_CMD +%s%N)
for i in $(seq 1 10); do
    curl -s "$API_URL/health" > /dev/null &
done
wait
db_end=$($DATE_CMD +%s%N)
db_duration=$(( (db_end - db_start) / 1000000 ))
echo -e "  10 parallel requests: ${GREEN}${db_duration}ms${NC}"

echo ""
echo -e "${YELLOW}[3/6] Compression Test${NC}"
no_compress=$(curl -s -o /dev/null -w "%{size_download}" "$API_URL/health")
with_compress=$(curl -s -o /dev/null -w "%{size_download}" -H "Accept-Encoding: gzip" "$API_URL/health")
if [ "$no_compress" -gt 0 ]; then
    savings=$(( (no_compress - with_compress) * 100 / no_compress ))
    echo -e "  Without compression: ${no_compress} bytes"
    echo -e "  With compression: ${GREEN}${with_compress} bytes${NC}"
    echo -e "  Savings: ${GREEN}${savings}%${NC}"
else
    echo -e "  ${YELLOW}Skipped (no response data)${NC}"
fi

echo ""
echo -e "${YELLOW}[4/6] Cache Performance${NC}"
# First request (MISS)
start=$($DATE_CMD +%s%N)
curl -s "$API_URL/api/projects" > /dev/null 2>&1
end=$($DATE_CMD +%s%N)
first_time=$(( (end - start) / 1000000 ))

# Second request (HIT)
start=$($DATE_CMD +%s%N)
curl -s "$API_URL/api/projects" > /dev/null 2>&1
end=$($DATE_CMD +%s%N)
second_time=$(( (end - start) / 1000000 ))

echo -e "  First request (MISS): ${first_time}ms"
echo -e "  Second request (HIT): ${GREEN}${second_time}ms${NC}"
if [ "$first_time" -gt 0 ]; then
    speedup=$(( first_time / second_time ))
    echo -e "  Cache speedup: ${GREEN}${speedup}x${NC}"
fi

echo ""
echo -e "${YELLOW}[5/6] Go Deploy Worker${NC}"
if [ -f "./bin/nidus-deploy-worker" ]; then
    binary_size=$(ls -lh ./bin/nidus-deploy-worker | awk '{print $5}')
    echo -e "  Binary size: ${GREEN}${binary_size}${NC}"
    
    # Memory usage
    ./bin/nidus-deploy-worker &
    worker_pid=$!
    sleep 1
    memory=$(ps -o rss= -p $worker_pid 2>/dev/null | awk '{print $1/1024 "MB"}')
    kill $worker_pid 2>/dev/null
    echo -e "  Memory usage: ${GREEN}${memory}${NC}"
else
    echo -e "  ${YELLOW}Binary not found${NC}"
fi

echo ""
echo -e "${YELLOW}[6/6] Throughput Test${NC}"
throughput=$(measure_throughput 100 "$API_URL/health" 10)
echo -e "  100 requests (10 concurrent): ${GREEN}${throughput} req/s${NC}"

echo ""
echo "========================================="
echo "  Results Summary"
echo "========================================="
echo ""
echo -e "  API Response:    ${GREEN}${api_time}ms${NC}"
echo -e "  Parallel (10):   ${GREEN}${db_duration}ms${NC}"
echo -e "  Cache Speedup:   ${GREEN}${speedup:-1}x${NC}"
echo -e "  Throughput:      ${GREEN}${throughput} req/s${NC}"
echo -e "  Compression:     ${GREEN}${savings:-0}% savings${NC}"
echo ""

# Save results
cat > "$RESULTS_FILE" << EOF
{
  "timestamp": "$(date -u +%Y-%m-%dT%H:%M:%SZ)",
  "api_response_ms": $api_time,
  "parallel_requests_ms": $db_duration,
  "cache_speedup": ${speedup:-1},
  "throughput_rps": $throughput,
  "compression_savings_percent": ${savings:-0}
}
EOF
echo -e "Results saved to: ${GREEN}${RESULTS_FILE}${NC}"
