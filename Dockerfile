# Build stage
FROM golang:1.25-alpine AS builder

WORKDIR /src

RUN apk add --no-cache git ca-certificates

COPY go.mod go.sum ./
COPY vendor ./vendor

COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build -mod=vendor -ldflags="-s -w" -o /out/dude-mcp ./cmd/dude-mcp

# Runtime stage
FROM alpine:3.20

RUN apk add --no-cache ca-certificates tzdata

WORKDIR /app

COPY --from=builder /out/dude-mcp ./dude-mcp
COPY --from=builder /src/internal/domain/dude_mcp/view ./internal/domain/dude_mcp/view
COPY --from=builder /src/config ./config

EXPOSE 8080 8081 8082

CMD ["./dude-mcp", "server"]
