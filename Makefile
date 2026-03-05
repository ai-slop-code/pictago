# Pictago — image host build and run
BINARY := pictago
# -s -w: strip symbol table and DWARF (smaller binary)
# -trimpath: reproducible builds, strip file paths
LDFLAGS := -s -w

.PHONY: build build-release run test cover clean deps fmt vet lint vulncheck help

# Default target: build the binary (with debug info for development)
build:
	go build -o $(BINARY) .

# Release build: smaller binary, no debug info, trimpath. Optional: make build-release VERSION=0.1.0
build-release:
	go build -trimpath -ldflags "$(LDFLAGS) -X main.version=$(VERSION)" -o $(BINARY) .

# Run: build (if needed) and start the server
run: build
	./$(BINARY)

# Run tests
test:
	go test ./...

# Run tests with coverage (writes coverage.out; view with: go tool cover -html=coverage.out)
cover:
	go test -coverprofile=coverage.out ./...
	go tool cover -func=coverage.out

# Format code (gofmt -s)
fmt:
	gofmt -s -w .

# Run go vet
vet:
	go vet ./...

# Run golangci-lint (install: go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest)
lint:
	golangci-lint run ./...

# Run govulncheck for known vulnerabilities (install: go install golang.org/x/vuln/cmd/govulncheck@latest)
vulncheck:
	govulncheck ./...

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
	@echo "  make run           - build and run the server"
	@echo "  make test         - run tests"
	@echo "  make cover        - run tests with coverage report"
	@echo "  make fmt          - format code (gofmt -s -w)"
	@echo "  make vet          - run go vet"
	@echo "  make lint         - run golangci-lint"
	@echo "  make vulncheck    - run govulncheck"
	@echo "  make deps         - download and tidy Go modules"
	@echo "  make clean        - remove binary and clean cache"
	@echo "  make help         - show this help"
