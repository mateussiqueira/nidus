#!/bin/bash
set -e
API_URL="${API_URL:-http://localhost:3001}"
RED="\\033[0;31m"; GREEN="\\033[0;32m"; YELLOW="\\033[1;33m"; NC="\\033[0m"
echo "========================================="
echo "  StackRun Performance Benchmark Suite"
echo "========================================="
echo ""

measure_ms() {
    local start=$(date +%s%3N 2>/dev/null || python3 -c "import time;print(int(time.time()*1000))")
    eval "$2" > /dev/null 2>&1
    local end=$(date +%s%3N 2>/dev/null || python3 -c "import time;print(int(time.time()*1000))")
    echo $(( end - start ))
}

echo -e "${YELLOW}[1/6] API Response Time${NC}"
HEALTH_TIME=$(measure_ms "curl -sf $API_URL/health")
echo "  Health endpoint: ${GREEN}${HEALTH_TIME}ms${NC}"
METRICS_TIME=$(measure_ms "curl -sf $API_URL/api/metrics")
echo "  Metrics endpoint: ${GREEN}${METRICS_TIME}ms${NC}"

echo ""
echo -e "${YELLOW}[2/6] Deploy Queue Speed${NC}"
TOKEN=$(curl -sf -X POST $API_URL/api/auth/login -H Content-Type:application/json -d "{\"email\":\"demo@stackrun.dev\",\"password\":\"demo123456\"}" | python3 -c "import sys,json;print(json.load(sys.stdin).get(\"token\",\"\"))" 2>/dev/null)

# Create test project
PROJ=$(curl -sf $API_URL/api/projects -H "Authorization: Bearer $TOKEN" | python3 -c "import sys,json;d=json.load(sys.stdin);print(d[0][\"id\"] if d else \"\")" 2>/dev/null)
if [ -z "$PROJ" ]; then
  PROJ=$(curl -sf -X POST $API_URL/api/projects -H "Content-Type:application/json" -H "Authorization: Bearer $TOKEN" -d "{\"name\":\"bench-test\",\"slug\":\"bench-test\"}" | python3 -c "import sys,json;print(json.load(sys.stdin)[\"id\"])" 2>/dev/null)
fi

DEPLOY_TIME=$(measure_ms "curl -sf -X POST \"$API_URL/api/projects/$PROJ/deploy\" -H \"Authorization: Bearer $TOKEN\"")
echo "  Deploy enqueue: ${GREEN}${DEPLOY_TIME}ms${NC}"

sleep 6
STATUS=$(curl -sf "$API_URL/api/projects/$PROJ/deployments" -H "Authorization: Bearer $TOKEN" | python3 -c "import sys,json;d=json.load(sys.stdin);print(d[0].get(\"status\",\"\") if d else \"\")" 2>/dev/null)
echo "  Deploy result: ${GREEN}${STATUS}${NC}"

echo ""
echo -e "${YELLOW}[3/6] Memory Usage${NC}"
free -h | head -2
echo ""
echo -e "${YELLOW}[4/6] Docker Stats${NC}"
docker ps --format "  {{.Names}}: {{.Status}}" | head -10

echo ""
echo -e "${YELLOW}[5/6] PM2 Processes${NC}"
pm2 jlist 2>/dev/null | python3 -c "
import sys,json
apps=json.load(sys.stdin)
for a in apps:
    n=a.get(\"name\",\"\")
    m=a.get(\"monit\",{})
    print(f\"  {n}: {m.get(\"memory\",0)/1024/1024:.0f}MB\")
" 2>/dev/null || pm2 status 2>/dev/null | grep stackrun

echo ""
echo -e "${YELLOW}[6/6] Disk${NC}"
df -h / | tail -1
echo ""
echo -e "${GREEN}Benchmark complete${NC}"
