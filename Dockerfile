## Multi-stage Dockerfile for Pictago
##
## Build stage: compile static Linux binary with version support.
FROM golang:1.22-alpine AS build

WORKDIR /app

# Install build tools (git is often needed for Go module fetching).
RUN apk add --no-cache ca-certificates git

# Cache dependencies first.
COPY go.mod go.sum ./
RUN go mod download

# Copy the rest of the source.
COPY . .

# Optional build-time version (override when building the image).
ARG VERSION=dev

# Build a static binary for Linux.
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags "-s -w -X main.version=${VERSION}" -o pictago .

## Runtime stage: minimal image.
FROM alpine:3.20

# Create non-root user.
RUN addgroup -S app && adduser -S app -G app

WORKDIR /app

# Copy binary from build stage.
COPY --from=build /app/pictago /app/pictago

# Default data directory (can be overridden with DATA_DIR env).
ENV DATA_DIR=/data

# Create data directory and give ownership to app user.
RUN mkdir -p "${DATA_DIR}" && chown -R app:app "${DATA_DIR}"

VOLUME ["/data"]

EXPOSE 8080

USER app

ENTRYPOINT ["/app/pictago"]
