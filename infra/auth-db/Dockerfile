# Dude MCP Provider — Postgres only.
# Schema + seed are applied by Go: go run ./cmd/migrate (GORM AutoMigrate).

FROM postgres:16-alpine

LABEL org.opencontainers.image.title="dude_mcp_db"
LABEL org.opencontainers.image.description="Postgres for Dude MCP identity provider (schema via GORM)"

HEALTHCHECK --interval=10s --timeout=5s --start-period=15s --retries=5 \
  CMD pg_isready -U "$POSTGRES_USER" -d "$POSTGRES_DB" || exit 1
