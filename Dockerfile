# Multi-stage Dockerfile for mizu CLI
# For local development builds

# Build stage
FROM golang:1.25-alpine AS builder

# Install build dependencies
RUN apk add --no-cache git ca-certificates tzdata

WORKDIR /src

# Cache dependencies
COPY go.mod go.sum* ./
RUN go mod download

# Copy source
COPY . .

# Build arguments for version injection
ARG VERSION=dev
ARG COMMIT=unknown
ARG BUILD_TIME=unknown

# Build binary
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath \
    -ldflags="-s -w \
        -X 'github.com/go-mizu/mizu/cli.Version=${VERSION}' \
        -X 'github.com/go-mizu/mizu/cli.Commit=${COMMIT}' \
        -X 'github.com/go-mizu/mizu/cli.BuildTime=${BUILD_TIME}'" \
    -o /mizu ./cmd/mizu

# Runtime stage
FROM alpine:3.21

# Install runtime dependencies
RUN apk add --no-cache ca-certificates tzdata git

# Create non-root user
RUN adduser -D -u 1000 mizu
USER mizu

# Copy binary from builder
COPY --from=builder /mizu /usr/local/bin/mizu

# Set working directory
WORKDIR /workspace

ENTRYPOINT ["/usr/local/bin/mizu"]
CMD ["--help"]
