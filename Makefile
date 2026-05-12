SERVICES := order-service notifier

.PHONY: build test tidy lint \
        migrate-up migrate-down \
        up down logs ps \
        build-order-service build-notifier \
        run-order-service run-notifier

# ── Local ────────────────────────────────────────────────────────────────────

build: ## Build all services
	@for svc in $(SERVICES); do \
		echo "→ building $$svc"; \
		cd $$svc && go build ./... && cd ..; \
	done

test: ## Run tests for all services
	@for svc in $(SERVICES); do \
		echo "→ testing $$svc"; \
		cd $$svc && go test ./... && cd ..; \
	done

tidy: ## Run go mod tidy for all services
	@for svc in $(SERVICES); do \
		echo "→ tidy $$svc"; \
		cd $$svc && go mod tidy && cd ..; \
	done

lint: ## Run golangci-lint for all services (requires golangci-lint)
	@for svc in $(SERVICES); do \
		echo "→ lint $$svc"; \
		cd $$svc && golangci-lint run ./... && cd ..; \
	done

run-order-service: ## Run order-service locally
	cd order-service && go run ./cmd/

run-notifier: ## Run notifier locally
	cd notifier && go run ./cmd/

migrate-up: ## Apply all pending migrations (requires running postgres)
	cd order-service && go run ./cmd/migrator/ -direction=up

migrate-down: ## Rollback all migrations (requires running postgres)
	cd order-service && go run ./cmd/migrator/ -direction=down

# ── Docker ───────────────────────────────────────────────────────────────────

up: ## Start all services via docker-compose (uses existing images; run `make up-build` after code changes)
	docker compose up -d

up-build: ## Rebuild images and start all services
	docker compose up --build -d

up-infra: ## Start only infrastructure (postgres + kafka + mailhog)
	docker compose up postgres kafka mailhog -d

down: ## Stop and remove containers
	docker compose down

down-v: ## Stop and remove containers + volumes
	docker compose down -v

logs: ## Tail logs from all containers
	docker compose logs -f

logs-order: ## Tail logs from order-service
	docker compose logs -f order-service

logs-notifier: ## Tail logs from notifier
	docker compose logs -f notifier

ps: ## Show running containers
	docker compose ps

restart: ## Restart all services (without rebuilding)
	docker compose restart order-service notifier

# ── Help ─────────────────────────────────────────────────────────────────────

help: ## Show this help
	@grep -E '^[a-zA-Z_-]+:.*##' $(MAKEFILE_LIST) | \
		awk 'BEGIN {FS = ":.*##"}; {printf "  \033[36m%-20s\033[0m %s\n", $$1, $$2}'

.DEFAULT_GOAL := help
