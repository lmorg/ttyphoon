# Variables
GO_FLAGS=-v
MXTTY_WINDOW?="markdown"
export MXTTY_WINDOW

# Build variables that can be overridden
BRANCH=$(shell git rev-parse --abbrev-ref HEAD || echo "unknown")
BUILD_DATE=$(shell date -u '+%Y-%m-%d_%H:%M:%S' || echo "unknown")
LDFLAGS=-ldflags "-X github.com/lmorg/ttyphoon/app.branch=${BRANCH} -X github.com/lmorg/ttyphoon/app.buildDate=${BUILD_DATE}"

# Build the binary
.PHONY: build
build: generate
	wails build

# Run the application
.PHONY: run-darwin
run-darwin: build
	unset MXTTY_WINDOW; ./build/bin/TTYphoon.app/Contents/MacOS/ttyphoon

# Test
.PHONY: test
test: generate
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
	export MXTTY_WINDOW
	@find . -name "*.go" -exec grep "//go:build" {} \; \
	| grep -v -E '(ignore|js|windows|linux|darwin|plan9|solaris|freebsd|openbsd|netbsd|dragonfly|aix)' \
	| sed -e 's,//go:build ,,;s,!,,;' \
	| sort -u
	@echo "sqlite_omit_load_extension\nosusergo\nnetgo"

# readline package development
local_readline  = local/readline
remote_readline = lmorg/readline

.PHONY: 
local-dev-readline:
ifneq "$(wildcard $(local_readline)/.)" ""
	cd $(local_readline)
	git pull
else
	@mkdir -p local
	git clone git@github.com:$(remote_readline).git $(local_readline)
endif
	cd $(local_readline)
	go mod edit -replace "github.com/$(remote_readline)/v4=./$(local_readline)"
	go mod tidy
	@echo ""
	@echo "Before you push any changes of Ttyphoon, you will need to run:"
	@echo "    make remote-readline"

.PHONY:
remote-readline:
	go mod edit -dropreplace=github.com/$(remote_readline)/v4
	go mod tidy

# glamour package development

local_glamour  = local/glamour
remote_glamour = charmbracelet/glamour

.PHONY: 
local-dev-glamour:
ifneq "$(wildcard $(local_glamour)/.)" ""
	cd $(local_glamour)
	git pull
else
	@mkdir -p local
	git clone git@github.com:$(remote_glamour).git $(local_glamour)
endif
	cd $(local_glamour)
	go mod edit -replace "github.com/$(remote_glamour)/v4=./$(local_glamour)"
	go mod tidy
	@echo ""
	@echo "Before you push any changes of Ttyphoon, you will need to run:"
	@echo "    make remote-readline"

.PHONY:
remote-glamour:
	go mod edit -dropreplace=github.com/$(remote_glamour)/v4
	go mod tidy

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
