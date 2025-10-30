FROM mirror.gcr.io/library/golang:1.24.3-alpine AS builder
WORKDIR /src

RUN apk add --no-cache git ca-certificates tzdata

ENV GOPROXY=https://proxy.golang.org,direct \
    GOSUMDB=sum.golang.org \
    GO111MODULE=on

COPY go.mod go.sum ./
RUN set -eux; for i in 1 2 3; do go mod download && break || (echo "retry $i"; sleep 3); done

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /out/app ./cmd/server

FROM mirror.gcr.io/library/alpine:3.20
RUN adduser -D -g '' app
USER app
WORKDIR /app

COPY --from=builder /out/app /app/app

EXPOSE 8080
HEALTHCHECK --interval=5s --timeout=3s --retries=12 CMD wget -qO- http://localhost:8080/health >/dev/null 2>&1 || exit 1

ENTRYPOINT ["/app/app"]
