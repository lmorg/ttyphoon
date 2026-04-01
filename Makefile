# Variables
GO_FLAGS=-v

# Build variables that can be overridden
APP_NAME=$(shell jq -r '.name' wails.json || echo "unknown")
TAG_LINE=$(shell jq -r '.info.comments' wails.json || echo "unknown")
VERSION=$(shell jq -r '.info.productVersion' wails.json || echo "unknown")
BRANCH=$(shell git rev-parse --abbrev-ref HEAD || echo "unknown")
BUILD_DATE=$(shell date -u '+%Y-%m-%d_%H:%M:%S' || echo "unknown")
COPYRIGHT=$(shell jq -r '.info.copyright' wails.json || echo "unknown")

PKG_PATH="github.com/lmorg/ttyphoon"
LDFLAGS="-X '${PKG_PATH}/app.name=${APP_NAME}' -X '${PKG_PATH}/app.tagLine=${TAG_LINE}' -X '${PKG_PATH}/app.version=${VERSION}' -X '${PKG_PATH}/app.branch=${BRANCH}' -X '${PKG_PATH}/app.buildDate=${BUILD_DATE}' -X '${PKG_PATH}/app.copyright=${COPYRIGHT}'"


# Build the binary
.PHONY: build
build: generate
	wails build -ldflags ${LDFLAGS}

# Clean build the binary
.PHONY: clean
clean: generate
	wails build -clean -ldflags ${LDFLAGS}

# Run the application
.PHONY: run-darwin
run-darwin: build
	./build/bin/TTYphoon.app/Contents/MacOS/ttyphoon

.PHONY: run-webkit
run-webkit:
	wails dev


# Test
.PHONY: test
test: generate
	npm --prefix frontend run test:run
	go test ./... -count 1 -race -covermode=atomic


# Benchmark
.PHONY: bench
bench:
	go test -bench=. -benchmem ./...

# Update dependencies
.PHONY: update-deps
update-deps:
	go get -u ./...
	go mod tidy

# Generate code
.PHONY: generate
generate:
	@echo "Running code generation..."
	go generate ./...
	wails generate module

# List available build tags
.PHONY: list-build-tags
list-build-tags:
	@find . -name "*.go" -exec grep "//go:build" {} \; \
	| grep -v -E '(ignore|js|windows|linux|darwin|plan9|solaris|freebsd|openbsd|netbsd|dragonfly|aix)' \
	| sed -e 's,//go:build ,,;s,!,,;' \
	| sort -u
	@echo "sqlite_omit_load_extension\nosusergo\nnetgo"

# Help
.PHONY: help
help:
	@echo "Available commands:"
	@echo "  make build           - Build TTYphoon"
	@echo "  make run-darwin      - Build and run macOS binary"
	@echo '  make list-build-tags - list tags supported by `$$BUILD_TAGS`'
	@echo ""
	@echo "Development tools:"
#	@echo "  make build-dev       - Build with profiling and debug symbols"
#	@echo "  make run             - Build and run a dev build of Ttyphoon"
	@echo "  make test            - Run tests"
#	@echo "  make bench           - Run benchmarks"
