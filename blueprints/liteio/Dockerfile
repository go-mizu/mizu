# Build stage
FROM golang:1.26-alpine AS builder

ARG VERSION=dev
ARG COMMIT=unknown
ARG BUILD_TIME=

WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath \
    -ldflags "-s -w -X main.Version=${VERSION} -X main.Commit=${COMMIT} -X main.BuildTime=${BUILD_TIME}" \
    -o /liteio ./cmd/liteio

# Runtime stage
FROM scratch

COPY --from=builder /liteio /usr/local/bin/liteio

EXPOSE 9000
VOLUME /data

ENTRYPOINT ["liteio"]
CMD ["--port", "9000", "--host", "0.0.0.0", "--data-dir", "/data"]
