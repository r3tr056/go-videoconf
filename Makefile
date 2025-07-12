# VideoConf Makefile

.PHONY: help build test clean dev deploy docker-build docker-up docker-down

help: ## Show this help message
	@echo 'Usage: make [target]'
	@echo ''
	@echo 'Available targets:'
	@awk 'BEGIN {FS = ":.*?## "} /^[a-zA-Z_-]+:.*?## / {printf "  %-15s %s\n", $$1, $$2}' $(MAKEFILE_LIST)

build: ## Build all services
	@echo "Building signalling server..."
	cd server/signalling-server && go build -o signalling-server
	@echo "Building users service..."
	cd server/users-service && go build -o users-service
	@echo "Building SDK..."
	cd sdk && npm install && npm run build
	@echo "All services built successfully!"

test: ## Run tests for all services
	@echo "Testing signalling server..."
	cd server/signalling-server && go test -v
	@echo "Testing users service..."
	cd server/users-service && go test -v
	@echo "All tests passed!"

clean: ## Clean build artifacts
	@echo "Cleaning build artifacts..."
	rm -f server/signalling-server/signalling-server
	rm -f server/users-service/users-service
	rm -rf sdk/dist
	rm -rf client/build
	rm -rf client/node_modules
	rm -rf sdk/node_modules
	@echo "Clean completed!"

dev: ## Start development environment
	@echo "Starting development environment..."
	docker-compose up --build

docker-build: ## Build Docker images
	@echo "Building Docker images..."
	docker-compose build

docker-up: ## Start Docker containers
	@echo "Starting Docker containers..."
	docker-compose up -d

docker-down: ## Stop Docker containers
	@echo "Stopping Docker containers..."
	docker-compose down

docker-logs: ## View Docker logs
	docker-compose logs -f

deploy-k8s: ## Deploy to Kubernetes
	@echo "Deploying to Kubernetes..."
	kubectl apply -f .deployment/mongo-deployment.yml
	kubectl apply -f .deployment/server-deployment.yml
	kubectl apply -f .deployment/client-deployment.yml
	kubectl apply -f .deployment/ingress.yml
	@echo "Deployment completed!"

undeploy-k8s: ## Remove from Kubernetes
	@echo "Removing from Kubernetes..."
	kubectl delete -f .deployment/
	@echo "Undeployment completed!"

install-deps: ## Install dependencies
	@echo "Installing Go dependencies..."
	cd server/signalling-server && go mod tidy
	cd server/users-service && go mod tidy
	@echo "Installing SDK dependencies..."
	cd sdk && npm install
	@echo "Installing client dependencies..."
	cd client && npm install
	@echo "Dependencies installed!"

lint: ## Run linters
	@echo "Running Go linters..."
	cd server/signalling-server && go vet ./...
	cd server/users-service && go vet ./...
	@echo "Linting completed!"

format: ## Format code
	@echo "Formatting Go code..."
	cd server/signalling-server && go fmt ./...
	cd server/users-service && go fmt ./...
	@echo "Formatting completed!"

check-health: ## Check service health
	@echo "Checking service health..."
	curl -f http://localhost:8080/health || echo "Signalling server not healthy"
	curl -f http://localhost:8081/health || echo "Users service not healthy"
	curl -f http://localhost:80 || echo "Load balancer not healthy"

setup: install-deps build ## Full setup (install deps and build)
	@echo "Setup completed!"

all: clean setup test ## Clean, setup, and test everything
	@echo "All tasks completed successfully!"