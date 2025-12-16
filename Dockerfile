# Multi-stage Dockerfile for mizu CLI
# Local development builds

# Build stage
FROM golang:1.25-alpine AS builder

# Build dependencies
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
    -X github.com/go-mizu/mizu/cli.Version=${VERSION} \
    -X github.com/go-mizu/mizu/cli.Commit=${COMMIT} \
    -X github.com/go-mizu/mizu/cli.BuildTime=${BUILD_TIME}" \
  -o /out/mizu ./cmd/mizu

# Runtime stage
FROM alpine:3.23

# Runtime dependencies (keep minimal)
RUN apk add --no-cache ca-certificates tzdata

# Create non-root user
RUN adduser -D -u 1000 mizu

# Copy binary
COPY --from=builder /out/mizu /usr/local/bin/mizu

USER mizu
WORKDIR /workspace

ENTRYPOINT ["/usr/local/bin/mizu"]
CMD ["--help"]
