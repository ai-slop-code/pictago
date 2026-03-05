# Pictago — image host build and run
BINARY := pictago
# -s -w: strip symbol table and DWARF (smaller binary)
# -trimpath: reproducible builds, strip file paths
LDFLAGS := -s -w

.PHONY: build build-release run test clean deps help

# Default target: build the binary (with debug info for development)
build:
	go build -o $(BINARY) .

# Release build: smaller binary, no debug info, trimpath
build-release:
	go build -trimpath -ldflags "$(LDFLAGS)" -o $(BINARY) .

# Run: build (if needed) and start the server
run: build
	./$(BINARY)

# Run tests
test:
	go test ./...

# Download and tidy dependencies
deps:
	go mod download
	go mod tidy

# Remove built binary and clean Go cache for this module
clean:
	rm -f $(BINARY)
	go clean

# Show available targets
help:
	@echo "Pictago — targets:"
	@echo "  make build         - build binary ($(BINARY))"
	@echo "  make build-release - build smaller release binary (strip + trimpath)"
	@echo "  make run          - build and run the server"
	@echo "  make test         - run tests"
	@echo "  make deps         - download and tidy Go modules"
	@echo "  make clean        - remove binary and clean cache"
	@echo "  make help         - show this help"
