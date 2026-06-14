# Build stage
FROM golang:1.25-alpine AS builder

WORKDIR /src

RUN apk add --no-cache git ca-certificates

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /out/triov2 ./cmd/triov2

# Runtime stage
FROM alpine:3.20

RUN apk add --no-cache ca-certificates tzdata

WORKDIR /app

COPY --from=builder /out/triov2 ./triov2
COPY --from=builder /src/internal/domain/trio_mcp/view ./internal/domain/trio_mcp/view
COPY --from=builder /src/config ./config

EXPOSE 8080 8081 8082

CMD ["./triov2", "server"]
