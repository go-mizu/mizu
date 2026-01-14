# Build stage
FROM golang:1.25-alpine AS builder

WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o /liteio ./cmd/liteio

# Runtime stage
FROM scratch

COPY --from=builder /liteio /usr/local/bin/liteio

EXPOSE 9000
VOLUME /data

ENTRYPOINT ["liteio"]
CMD ["--port", "9000", "--host", "0.0.0.0", "--data-dir", "/data"]
