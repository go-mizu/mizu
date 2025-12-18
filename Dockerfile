# Multi-stage Dockerfile for mizu CLI
# Local development builds

# Build stage
FROM golang:1.25-alpine AS builder

# Build dependencies
RUN apk add --no-cache git ca-certificates tzdata

# Disable go workspace to avoid conflicts with cmd/ module
ENV GOWORK=off

WORKDIR /src

# Copy source (includes cmd/go.mod with replace directive)
COPY . .

# Build arguments for version injection
ARG VERSION=dev
ARG COMMIT=unknown
ARG BUILD_TIME=unknown

# Build binary from cmd/ module
WORKDIR /src/cmd
RUN go mod download
RUN CGO_ENABLED=0 GOOS=linux GOWORK=off go build -trimpath \
  -ldflags="-s -w \
    -X github.com/go-mizu/mizu/cmd/cli.Version=${VERSION} \
    -X github.com/go-mizu/mizu/cmd/cli.Commit=${COMMIT} \
    -X github.com/go-mizu/mizu/cmd/cli.BuildTime=${BUILD_TIME}" \
  -o /out/mizu ./mizu

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
