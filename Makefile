.PHONY: all build run test clean docker docker-multi frontend backend download-db

# Variables
BINARY_NAME=ipwhere
DOCKER_IMAGE=ipwhere
DOCKER_TAG=latest
DATA_DIR=data
CITY_DB=$(DATA_DIR)/dbip-city-lite.mmdb
ASN_DB=$(DATA_DIR)/dbip-asn-lite.mmdb
MMDB_RELEASE_URL=https://github.com/Shoyu-Dev/mmdb-latest/releases/download/dbip-latest

all: frontend backend

# Build frontend
frontend:
	cd web && npm ci && npm run build
	rm -rf cmd/ipwhere/static/*
	cp -r web/dist/* cmd/ipwhere/static/

# Build backend
backend:
	go build -o $(BINARY_NAME) ./cmd/ipwhere

# Build everything
build: frontend backend

# Run locally (requires MMDB files in data/)
run: build
	./$(BINARY_NAME)

# Run tests
test:
	go test -v ./...
	cd web && npm test

# Run Go tests only
test-go:
	go test -v ./...

# Run frontend tests only
test-frontend:
	cd web && npm test

# Clean build artifacts
clean:
	rm -f $(BINARY_NAME)
	rm -rf web/dist
	rm -rf cmd/ipwhere/static/*
	touch cmd/ipwhere/static/.gitkeep

# Build Docker image (single architecture)
# Downloads databases first to avoid redundant downloads in Docker
docker: download-db
	docker build -t $(DOCKER_IMAGE):$(DOCKER_TAG) .

# Build Docker image for multiple architectures
# Downloads databases once before build, avoiding duplicate downloads per arch
docker-multi: download-db
	docker buildx build --platform linux/amd64,linux/arm64 -t $(DOCKER_IMAGE):$(DOCKER_TAG) .

# Build and push multi-arch image
docker-push: download-db
	docker buildx build --platform linux/amd64,linux/arm64 -t $(DOCKER_IMAGE):$(DOCKER_TAG) --push .

# Run Docker container
docker-run:
	docker run -p 8080:8080 $(DOCKER_IMAGE):$(DOCKER_TAG)

# Run Docker container in headless mode
docker-run-headless:
	docker run -p 8080:8080 -e HEADLESS=true $(DOCKER_IMAGE):$(DOCKER_TAG)

# Generate Swagger documentation
swagger:
	swag init -g cmd/ipwhere/main.go -o docs

# Download MMDB databases (used for local dev and Docker builds)
# Uses file-based targets to avoid re-downloading if files exist
# Requires: curl (available on macOS and most Linux distros)
$(DATA_DIR):
	mkdir -p $(DATA_DIR)

$(CITY_DB): | $(DATA_DIR)
	@command -v curl >/dev/null 2>&1 || { echo "Error: curl is required. Install with: apt install curl (Linux) or brew install curl (macOS)"; exit 1; }
	@echo "Downloading city database..."
	curl -fsSL "$(MMDB_RELEASE_URL)/dbip-city-lite.mmdb" -o $(CITY_DB)

$(ASN_DB): | $(DATA_DIR)
	@command -v curl >/dev/null 2>&1 || { echo "Error: curl is required. Install with: apt install curl (Linux) or brew install curl (macOS)"; exit 1; }
	@echo "Downloading ASN database..."
	curl -fsSL "$(MMDB_RELEASE_URL)/dbip-asn-lite.mmdb" -o $(ASN_DB)

download-db: $(CITY_DB) $(ASN_DB)
	@echo "MMDB databases ready in $(DATA_DIR)/"

# Development mode - run frontend dev server
dev-frontend:
	cd web && npm run dev

# Development mode - run backend (requires MMDB files)
dev-backend:
	go run ./cmd/ipwhere

# Help
help:
	@echo "Available targets:"
	@echo "  all              - Build frontend and backend"
	@echo "  build            - Build everything"
	@echo "  frontend         - Build frontend only"
	@echo "  backend          - Build backend only"
	@echo "  run              - Build and run locally"
	@echo "  test             - Run all tests"
	@echo "  test-go          - Run Go tests only"
	@echo "  test-frontend    - Run frontend tests only"
	@echo "  clean            - Clean build artifacts"
	@echo "  download-db      - Download MMDB databases (auto-run by docker targets)"
	@echo "  docker           - Build Docker image (downloads DBs first)"
	@echo "  docker-multi     - Build multi-arch Docker image (downloads DBs first)"
	@echo "  docker-push      - Build and push multi-arch image"
	@echo "  docker-run       - Run Docker container"
	@echo "  docker-run-headless - Run Docker container in headless mode"
	@echo "  swagger          - Generate Swagger documentation"
	@echo "  dev-frontend     - Run frontend dev server"
	@echo "  dev-backend      - Run backend in dev mode"
