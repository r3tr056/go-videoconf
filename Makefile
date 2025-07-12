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
	@echo "Building client..."
	cd client && npm install && npm run build
	@echo "All services built successfully!"

build-optimized: ## Build all services with optimizations
	@echo "Building optimized signalling server..."
	cd server/signalling-server && CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -ldflags '-extldflags "-static"' -o signalling-server
	@echo "Building optimized users service..."
	cd server/users-service && CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -ldflags '-extldflags "-static"' -o users-service
	@echo "Building optimized SDK..."
	cd sdk && npm ci --only=production && npm run build
	@echo "Building optimized client..."
	cd client && npm ci --only=production && npm run build
	@echo "All optimized services built successfully!"

test: ## Run tests for all services
	@echo "Testing signalling server..."
	cd server/signalling-server && go test -v ./...
	@echo "Testing users service..."
	cd server/users-service && go test -v ./...
	@echo "Testing SDK..."
	cd sdk && npm test
	@echo "All tests passed!"

test-comprehensive: ## Run comprehensive tests including stress tests
	@echo "Running comprehensive Go tests..."
	cd server/signalling-server && go test -v -race -coverprofile=coverage.out ./... && go test -v ./tests/...
	cd server/users-service && go test -v -race -coverprofile=coverage.out ./... && go test -v ./tests/...
	@echo "Running SDK tests with coverage..."
	cd sdk && npm run test:coverage
	@echo "All comprehensive tests passed!"

test-stress: ## Run stress tests (requires services to be running)
	@echo "Running stress tests..."
	cd server/signalling-server && go test -v -run="^TestStress" -timeout=10m ./tests/...
	cd server/users-service && go test -v -run="^TestLoad" -timeout=10m ./tests/...
	@echo "Stress tests completed!"

test-integration: ## Run integration tests (requires services to be running)
	@echo "Running integration tests..."
	go test -v -timeout=5m ./tests/integration/...
	@echo "Integration tests completed!"

test-all: test-comprehensive test-stress test-integration ## Run all tests including stress and integration tests

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

benchmark: ## Run performance benchmarks
	@echo "Running Go benchmarks..."
	cd server/signalling-server && go test -bench=. -benchmem ./...
	cd server/users-service && go test -bench=. -benchmem ./...
	@echo "Running SDK benchmarks..."
	cd sdk && npm run test -- --runInBand --testNamePattern="Performance"
	@echo "Benchmark completed!"

security-scan: ## Run security vulnerability scans
	@echo "Scanning Go dependencies..."
	cd server/signalling-server && go list -json -m all | docker run --rm -i sonatypecommunity/nancy:latest sleuth
	cd server/users-service && go list -json -m all | docker run --rm -i sonatypecommunity/nancy:latest sleuth
	@echo "Scanning npm dependencies..."
	cd sdk && npm audit --audit-level moderate
	cd client && npm audit --audit-level moderate
	@echo "Security scan completed!"

docs: ## Generate documentation
	@echo "Generating Go documentation..."
	cd server/signalling-server && godoc -http=:6060 &
	cd server/users-service && godoc -http=:6061 &
	@echo "Generating SDK documentation..."
	cd sdk && npm run docs
	@echo "Documentation generated!"

coverage: ## Generate test coverage reports
	@echo "Generating coverage reports..."
	cd server/signalling-server && go test -coverprofile=coverage.out ./... && go tool cover -html=coverage.out -o coverage.html
	cd server/users-service && go test -coverprofile=coverage.out ./... && go tool cover -html=coverage.out -o coverage.html
	cd sdk && npm run test:coverage
	@echo "Coverage reports generated!"

monitor: ## Start monitoring services
	@echo "Starting monitoring..."
	docker run -d --name prometheus -p 9090:9090 prom/prometheus
	docker run -d --name grafana -p 3001:3000 grafana/grafana
	@echo "Monitoring services started on ports 9090 (Prometheus) and 3001 (Grafana)"