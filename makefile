# Makefile für wp_plugin_release mit i18n Support
# 
# Unterstützte Targets:
# - build: Standard Build
# - build-all: Build für alle Plattformen
# - test: Tests ausführen  
# - clean: Build-Artefakte löschen
# - i18n-extract: Übersetzungskeys extrahieren
# - i18n-update: Übersetzungen aktualisieren
# - release: Release erstellen

# Variablen
BINARY_NAME=wp_plugin_release
VERSION=$(shell git describe --tags --always --dirty)
BUILD_TIME=$(shell date '+%Y-%m-%d %H:%M:%S')
GIT_COMMIT=$(shell git rev-parse HEAD)

# Go Build Flags
LDFLAGS=-ldflags "-X main.Version=${VERSION} -X 'main.BuildTime=${BUILD_TIME}' -X main.GitCommit=${GIT_COMMIT}"

# Plattformen für Cross-Compilation
PLATFORMS=windows/amd64 linux/amd64 darwin/amd64 darwin/arm64

# Standard Target
.PHONY: all
all: build

# Standard Build
.PHONY: build
build:
	@echo "Building ${BINARY_NAME} ${VERSION}..."
	go build ${LDFLAGS} -o bin/${BINARY_NAME}

# Build für alle Plattformen
.PHONY: build-all
build-all:
	@echo "Building for all platforms..."
	@for platform in $(PLATFORMS); do \
		GOOS=$${platform%/*} GOARCH=$${platform#*/} \
		go build ${LDFLAGS} \
			-o bin/${BINARY_NAME}-$${platform%/*}-$${platform#*/}$(if $(findstring windows,$$platform),.exe,) .; \
		echo "Built for $$platform"; \
	done

# Tests ausführen
.PHONY: test
test:
	@echo "Running tests..."
	go test -v ./...

# Build-Artefakte löschen
.PHONY: clean
clean:
	@echo "Cleaning build artifacts..."
	rm -rf bin/
	rm -f *.log
	rm -f *.bak

# Übersetzungskeys aus dem Code extrahieren
.PHONY: i18n-extract
i18n-extract:
	@echo "Extracting translation keys from source code..."
	@mkdir -p locales
	@echo "Scanning Go files for t() calls..."
	@grep -rohE 't\("([^"]+)"' . --include="*.go" | \
		sed 's/t("\([^"]*\)".*/"\1":/' | \
		sort | uniq > locales/keys_found.txt || true
	@echo "Translation keys extracted to locales/keys_found.txt"
	@echo "Please update locales/en.json and locales/de.json manually"

# Übersetzungen validieren
.PHONY: i18n-validate
i18n-validate:
	@echo "Validating translations..."
	@go run tools/validate_i18n.go

# Entwicklungsumgebung vorbereiten
.PHONY: setup
setup:
	@echo "Setting up development environment..."
	go mod tidy
	go mod download
	mkdir -p bin locales
	@if [ ! -f locales/en.json ]; then \
		echo "Creating default English translations..."; \
		echo '{}' > locales/en.json; \
	fi
	@if [ ! -f locales/de.json ]; then \
		echo "Creating default German translations..."; \
		echo '{}' > locales/de.json; \
	fi

# Release erstellen
.PHONY: release
release: clean build-all
	@echo "Creating release ${VERSION}..."
	@mkdir -p dist
	@for platform in $(PLATFORMS); do \
		BINARY=bin/${BINARY_NAME}-$${platform%/*}-$${platform#*/}$(if $(findstring windows,$$platform),.exe,); \
		if [ -f "$$BINARY" ]; then \
			tar -czf dist/${BINARY_NAME}-${VERSION}-$${platform%/*}-$${platform#*/}.tar.gz \
				-C bin $$(basename $$BINARY) \
				-C .. README.md LICENSE locales/; \
			echo "Created release package for $$platform"; \
		fi \
	done

# Lokalisierung testen (beide Sprachen)
.PHONY: test-i18n
test-i18n:
	@echo "Testing German localization..."
	LANG=de_DE.UTF-8 ./bin/${BINARY_NAME} --help || true
	@echo "Testing English localization..."
	LANG=en_US.UTF-8 ./bin/${BINARY_NAME} --help || true

# Git Pre-commit Hook installieren
.PHONY: install-hooks
install-hooks:
	@echo "Installing git pre-commit hooks..."
	@echo '#!/bin/sh' > .git/hooks/pre-commit
	@echo 'make test i18n-validate' >> .git/hooks/pre-commit
	@chmod +x .git/hooks/pre-commit
	@echo "Pre-commit hook installed"

# Dokumentation generieren
.PHONY: docs
docs:
	@echo "Generating documentation..."
	@mkdir -p docs
	go doc ./... > docs/api.md
	@echo "Documentation generated in docs/"

# Docker Image bauen
.PHONY: docker
docker:
	@echo "Building Docker image..."
	docker build -t ${BINARY_NAME}:${VERSION} -t ${BINARY_NAME}:latest .

# Dependencies aktualisieren
.PHONY: update-deps
update-deps:
	@echo "Updating dependencies..."
	go get -u ./...
	go mod tidy

# Code-Qualität prüfen
.PHONY: lint
lint:
	@echo "Running linters..."
	@if command -v golangci-lint >/dev/null 2>&1; then \
		golangci-lint run; \
	else \
		echo "golangci-lint not installed. Install with: go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest"; \
		go vet ./...; \
		go fmt ./...; \
	fi

# Hilfe anzeigen
.PHONY: help
help:
	@echo "Available targets:"
	@echo "  build          - Build the application"
	@echo "  build-all      - Build for all supported platforms"
	@echo "  test           - Run tests"
	@echo "  clean          - Clean build artifacts"
	@echo "  i18n-extract   - Extract translation keys from source"
	@echo "  i18n-validate  - Validate translations"
	@echo "  setup          - Setup development environment"
	@echo "  release        - Create release packages"
	@echo "  test-i18n      - Test localization"
	@echo "  install-hooks  - Install git hooks"
	@echo "  docs           - Generate documentation"
	@echo "  docker         - Build Docker image"
	@echo "  update-deps    - Update dependencies"
	@echo "  lint           - Run code linters
	@echo "  help           - Show this help message"