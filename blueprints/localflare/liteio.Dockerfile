# LiteIO - minimal scratch runtime

FROM golang:1.25-alpine3.23 AS builder

WORKDIR /build

COPY go.mod go.sum ./
RUN go mod download

COPY . .

ARG VERSION=dev
ARG COMMIT=unknown
ARG BUILD_TIME=unknown

RUN CGO_ENABLED=0 GOOS=linux go build \
  -trimpath \
  -ldflags="-s -w -X main.Version=${VERSION} -X main.Commit=${COMMIT} -X main.BuildTime=${BUILD_TIME}" \
  -o /liteio \
  ./cmd/liteio


FROM scratch

COPY --from=builder /liteio /liteio

EXPOSE 9000
VOLUME ["/data"]

ENTRYPOINT ["/liteio"]
CMD ["--data-dir", "/data", "--host", "0.0.0.0"]
