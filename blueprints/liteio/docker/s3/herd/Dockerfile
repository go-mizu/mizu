# Build stage
FROM golang:1.26-alpine AS builder

WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath \
    -ldflags "-s -w" \
    -o /herd ./cmd/herd

# Runtime stage — busybox for wget/du healthcheck
FROM busybox:1.36

COPY --from=builder /herd /usr/local/bin/herd

EXPOSE 9000
VOLUME /data

ENTRYPOINT ["herd"]
CMD ["-listen", ":9000", "-data-dir", "/data", "-no-log", "-stripes", "16", "-sync", "none", "-inline-kb", "8", "-prealloc", "1024"]
