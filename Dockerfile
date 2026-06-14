# Build stage
FROM golang:1.25-alpine AS builder

WORKDIR /src

RUN apk add --no-cache git ca-certificates

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /out/open-mcp ./cmd/open-mcp

# Runtime stage
FROM alpine:3.20

RUN apk add --no-cache ca-certificates tzdata

WORKDIR /app

COPY --from=builder /out/open-mcp ./open-mcp
COPY --from=builder /src/internal/domain/open_mcp/view ./internal/domain/open_mcp/view
COPY --from=builder /src/config ./config

EXPOSE 8080 8081 8082

CMD ["./open-mcp", "server"]
