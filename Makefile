# ─────────────────────────────────────────────────────────────────────────────
#  Quiz System – Makefile
# ─────────────────────────────────────────────────────────────────────────────

APP_CONTAINER  := quiz_app
MONGO_CONTAINER := quiz_mongo
BINARY         := ./bin/server
CMD_PATH       := ./cmd/server

.PHONY: help up down restart build logs logs-app logs-mongo \
        shell mongo-shell run test lint tidy clean nuke

# ── Default ───────────────────────────────────────────────────────────────────
help:
	@echo ""
	@echo "  Quiz System – available commands"
	@echo ""
	@echo "  Docker"
	@echo "  ──────────────────────────────────────────"
	@echo "  make up          Start containers (detached)"
	@echo "  make down        Stop containers"
	@echo "  make restart     Rebuild image and restart app"
	@echo "  make build       Rebuild image only (no start)"
	@echo "  make logs        Tail logs for all containers"
	@echo "  make logs-app    Tail app container logs"
	@echo "  make logs-mongo  Tail mongo container logs"
	@echo "  make shell       Open shell inside app container"
	@echo "  make mongo-shell Open mongo shell"
	@echo ""
	@echo "  Local development (no Docker)"
	@echo "  ──────────────────────────────────────────"
	@echo "  make run         Run the server locally"
	@echo "  make bin         Build binary to ./bin/server"
	@echo "  make test        Run all tests"
	@echo "  make lint        Run go vet"
	@echo "  make tidy        Tidy go modules"
	@echo ""
	@echo "  Cleanup"
	@echo "  ──────────────────────────────────────────"
	@echo "  make clean       Stop containers and remove images"
	@echo "  make nuke        Stop containers, remove images + volumes (⚠ wipes DB)"
	@echo ""

# ── Docker ────────────────────────────────────────────────────────────────────
up:
	docker compose up -d

down:
	docker compose down

restart:
	docker compose up -d --build

build:
	docker compose build

logs:
	docker compose logs -f

logs-app:
	docker compose logs -f $(APP_CONTAINER)

logs-mongo:
	docker compose logs -f $(MONGO_CONTAINER)

shell:
	docker exec -it $(APP_CONTAINER) sh

mongo-shell:
	docker exec -it $(MONGO_CONTAINER) mongosh

# ── Local development ─────────────────────────────────────────────────────────
run:
	go run $(CMD_PATH)

bin:
	@mkdir -p bin
	CGO_ENABLED=0 go build -ldflags="-w -s" -o $(BINARY) $(CMD_PATH)

test:
	go test ./... -v

lint:
	go vet ./...

tidy:
	go mod tidy

# ── Cleanup ───────────────────────────────────────────────────────────────────
clean:
	docker compose down --rmi local

nuke:
	docker compose down --rmi local -v
