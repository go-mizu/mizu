# Build stage
FROM golang:1.24-alpine AS builder

WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath \
    -ldflags "-s -w" \
    -o /herd ./cmd/herd

# Runtime stage — busybox for wget healthcheck
FROM busybox:1.36

COPY --from=builder /herd /usr/local/bin/herd

EXPOSE 9000
VOLUME /data

ENTRYPOINT ["herd"]
