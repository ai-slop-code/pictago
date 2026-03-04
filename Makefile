# Pictago — image host build and run
BINARY := pictago

.PHONY: build run test clean deps help

# Default target: build the binary
build:
	go build -o $(BINARY) .

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
	@echo "  make build   - build binary ($(BINARY))"
	@echo "  make run    - build and run the server"
	@echo "  make test   - run tests"
	@echo "  make deps   - download and tidy Go modules"
	@echo "  make clean  - remove binary and clean cache"
	@echo "  make help   - show this help"
