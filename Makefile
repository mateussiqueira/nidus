.PHONY: build start stop clean test

# Build all binaries
build:
	cd apps/api && CGO_ENABLED=0 GOOS=darwin GOARCH=arm64 go build -ldflags="-s -w" -o nidus-api .
	cd workers/deploy && CGO_ENABLED=0 GOOS=darwin GOARCH=arm64 go build -ldflags="-s -w" -o nidus-deploy-worker .
	cd apps/proxy && cargo build --release
	@echo "Build completo!"

# Start local
start:
	./start.sh local

# Start docker
docker:
	./start.sh docker

# Stop
stop:
	-lsof -ti:3001 | xargs kill -9 2>/dev/null
	-lsof -ti:3080 | xargs kill -9 2>/dev/null
	@echo "Serviços parados"

# Clean
clean:
	-lsof -ti:3001 | xargs kill -9 2>/dev/null
	-lsof -ti:3080 | xargs kill -9 2>/dev/null
	rm -f apps/api/nidus-api workers/deploy/nidus-deploy-worker
	rm -rf apps/proxy/target
	@echo "Limpeza completa"

# Test endpoints
test:
	@echo "=== Health ==="
	@curl -s http://localhost:3001/health | python3 -m json.tool
	@echo ""
	@echo "=== Register ==="
	@curl -s -X POST http://localhost:3001/api/auth/register \
		-H "Content-Type: application/json" \
		-d '{"email":"test@test.com","name":"Test","password":"test123"}' | python3 -m json.tool
	@echo ""
	@echo "=== Proxy Health ==="
	@curl -s http://localhost:3080/health | python3 -m json.tool

# Docker compose
up:
	docker compose up --build -d

down:
	docker compose down

logs:
	docker compose logs -f

ps:
	docker compose ps
