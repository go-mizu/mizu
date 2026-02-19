# Build Herd from source
FROM golang:1.24-alpine AS builder

WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -o /herd ./cmd/herd/

FROM alpine:latest
COPY --from=builder /herd /usr/local/bin/herd
ENTRYPOINT ["herd"]
