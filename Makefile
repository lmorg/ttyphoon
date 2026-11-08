# Build the binary
.PHONY: build
build:
	wails task common:update:build-assets
	wails build

.PHONY: build-flags
build-flags:
	@GOOS= GOARCH= go run ./build/buildflags

# Clean build the binary
.PHONY: clean
clean:
	rm -rf build/bin frontend/dist

# Generate code
.PHONY: generate
generate:
	@echo "Running code generation..."
	go generate ./...
	wails generate bindings

.PHONY: package
package:
	wails task common:update:build-assets
	wails package

# Run the application
.PHONY: run-darwin
run-darwin: generate package
	./build/bin/ttyphoon.app/Contents/MacOS/ttyphoon

.PHONY: run-webkit
run-webkit:
	wails dev -config ./build/config.yml -port 9245

# Test
.PHONY: test
test: generate
	npm --prefix frontend run test:run -- --reporter=dot
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
	@echo "  make build-flags     - Print application metadata linker flags"
	@echo "  make run-darwin      - Build and run macOS binary"
	@echo '  make list-build-tags - list tags supported by `$$BUILD_TAGS`'
	@echo ""
	@echo "Development tools:"
#	@echo "  make build-dev       - Build with profiling and debug symbols"
#	@echo "  make run             - Build and run a dev build of Ttyphoon"
	@echo "  make test            - Run tests"
#	@echo "  make bench           - Run benchmarks"
