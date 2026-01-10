.PHONY: run build clean deps build-x86 build-x86-linux build-x86-macos build-amd64-linux build-amd64-macos build-arm64-macos build-all help docker-build docker-build-amd64 docker-push docker-clean

# Variables
IMAGE_NAME ?= workspace-webhooks
IMAGE_TAG ?= latest
REGISTRY ?= localhost:5000

# Ensure go is in PATH - add common locations
export PATH := /opt/homebrew/bin:/usr/local/go/bin:$(PATH)

# Build the application (native architecture)
build:
	go build -v -o workspace-webhooks .

# Build for x86 (386) on current OS
build-x86:
	GOARCH=386 go build -v -o workspace-webhooks-x86 .

# Build for x86 (386) on Linux
build-x86-linux:
	GOOS=linux GOARCH=386 go build -v -o workspace-webhooks-x86-linux .

# Build for x86 (386) on macOS
build-x86-macos:
	GOOS=darwin GOARCH=386 go build -v -o workspace-webhooks-x86-macos .

# Build for x86_64 (amd64) on Linux
build-amd64-linux:
	GOOS=linux GOARCH=amd64 go build -v -o workspace-webhooks-amd64-linux .

# Build for x86_64 (amd64) on macOS
build-amd64-macos:
	GOOS=darwin GOARCH=amd64 go build -v -o workspace-webhooks-amd64-macos .

# Build for ARM64 on macOS
build-arm64-macos:
	GOOS=darwin GOARCH=arm64 go build -v -o workspace-webhooks-arm64-macos .

# Build all variants
build-all: build build-x86-linux build-amd64-linux build-amd64-macos build-arm64-macos
	@echo "All binaries built successfully!"
	@ls -lh workspace-webhooks*

# Run the application with example config
run: build
	./workspace-webhooks -c config.yaml

# Clean build artifacts
clean:
	rm -f workspace-webhooks workspace-webhooks-*

# Install dependencies
deps:
	go mod download
	go mod tidy

# Docker build for amd64
docker-build-amd64:
	@echo "Building Docker image for amd64: $(IMAGE_NAME):$(IMAGE_TAG)-amd64"
	docker build --platform linux/amd64 -t $(IMAGE_NAME):$(IMAGE_TAG)-amd64 .
	@echo "Docker image built: $(IMAGE_NAME):$(IMAGE_TAG)-amd64"

# Docker build (default platform)
docker-build:
	@echo "Building Docker image: $(IMAGE_NAME):$(IMAGE_TAG)"
	docker build -t $(IMAGE_NAME):$(IMAGE_TAG) .
	@echo "Docker image built: $(IMAGE_NAME):$(IMAGE_TAG)"

# Docker build all platforms (requires buildx)
docker-build-all:
	@echo "Building Docker image for multiple platforms"
	docker buildx build --platform linux/amd64,linux/arm64 -t $(IMAGE_NAME):$(IMAGE_TAG) .

# Show Docker images
docker-images:
	docker images | grep $(IMAGE_NAME)

# Clean Docker images
docker-clean:
	@echo "Removing Docker images for $(IMAGE_NAME)"
	docker rmi -f $$(docker images -q $(IMAGE_NAME)) 2>/dev/null || true
	@echo "Docker images cleaned"

# Show help
help:
	@echo "workspace-webhooks - Build targets"
	@echo ""
	@echo "Usage: make [target]"
	@echo ""
	@echo "Go Build Targets:"
	@echo "  build                Build for native OS/architecture"
	@echo "  build-x86            Build x86 (386) for current OS"
	@echo "  build-x86-linux      Build x86 (386) for Linux"
	@echo "  build-x86-macos      Build x86 (386) for macOS"
	@echo "  build-amd64-linux    Build x86_64 (amd64) for Linux"
	@echo "  build-amd64-macos    Build x86_64 (amd64) for macOS"
	@echo "  build-arm64-macos    Build ARM64 for macOS"
	@echo "  build-all            Build all Go variants"
	@echo ""
	@echo "Docker Targets:"
	@echo "  docker-build         Build Docker image for current platform"
	@echo "  docker-build-amd64   Build Docker image for amd64"
	@echo "  docker-build-all     Build Docker images for multiple platforms (requires buildx)"
	@echo "  docker-images        Show Docker images"
	@echo "  docker-clean         Remove Docker images"
	@echo ""
	@echo "General Targets:"
	@echo "  run                  Build and run with config.yaml"
	@echo "  clean                Remove all built binaries"
	@echo "  deps                 Download and tidy dependencies"
	@echo "  help                 Show this help message"
	@echo ""
	@echo "Docker Variables (can be overridden):"
	@echo "  IMAGE_NAME           Docker image name (default: workspace-webhooks)"
	@echo "  IMAGE_TAG            Docker image tag (default: latest)"
	@echo "  REGISTRY             Docker registry (default: localhost:5000)"
	@echo ""
	@echo "Example:"
	@echo "  make docker-build-amd64"
	@echo "  make docker-build IMAGE_TAG=v1.0.0"
	@echo "  make docker-build-amd64 IMAGE_NAME=myrepo/lark-webhook"
