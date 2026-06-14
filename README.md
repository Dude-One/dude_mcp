# Open MCP

Identity panel + **platform MCP server** for Open MCP domains.

The **panel** (`cmd/server`) handles login, sessions, and context notes.
The **MCP server** (`cmd/mcp`, planned) exposes Open MCP domain APIs and curated docs to IDE LLM clients (Cursor, Claude Desktop, Codex) over HTTP JSON-RPC 2.0.

---

## How it works

```
[Cursor / Claude IDE]
       │
       │  POST /mcp  (HTTP JSON-RPC 2.0 — one endpoint)
       ▼
[cmd/mcp — MCP server :8081]
       │
       ├──▶  GET http://panel:8080/swagger/doc.json   ← ID domain Swagger
       ├──▶  GET http://panel:8080/panel/me            ← ID domain API
       └──▶  GET http://sm.example.com/api/devices     ← Infra domain API
```

The IDE's LLM calls MCP **tools** in plain conversation. No terminal, no manual API calls.
MCP fetches Swagger dynamically from each registered domain — adding a new API requires only a registry entry, no MCP code change.

Domain registration: [`config/sources.yaml.example`](config/sources.yaml.example).
Full design: [`docs/PROJECT_CONTEXT_AI_DESIGN.md`](docs/PROJECT_CONTEXT_AI_DESIGN.md).

---

## Prerequisites

- Go 1.22+
- Docker + Docker Compose

---

## Quick start — panel

```powershell
copy .env.example .env
docker compose up -d open-mcp-db
go mod tidy
go run ./cmd/server
```

- **Panel UI:** http://localhost:8080/
- **Panel Swagger:** http://localhost:8080/swagger/index.html
- **Login:** `admin` / `admin`
- **Postgres:** `localhost:5433` / database `open_mcp_db`

Or use **Open MCP: Server** from Run and Debug (F5) after the DB container is up.

---

## Quick start — MCP server (planned)

```powershell
copy config/sources.yaml.example config/sources.yaml
# edit sources.yaml — set base_url, auth env vars
go run ./cmd/mcp
```

- **MCP endpoint:** http://localhost:8081/mcp
- **Transport:** HTTP Streamable (JSON-RPC 2.0) — not stdio

**IDE config** (add to Cursor / Claude Desktop MCP settings):

```json
{
  "mcpServers": {
    "trio": {
      "url": "http://localhost:8081/mcp",
      "headers": {
        "Authorization": "Bearer <your panel session token>"
      }
    }
  }
}
```

---

## Panel API

| Method | Path | Purpose |
|--------|------|---------|
| `POST` | `/panel/login` | Login; sets `open_mcp_session` cookie |
| `POST` | `/panel/signup` | Register on default tenant |
| `POST` | `/panel/logout` | Revoke session |
| `GET` | `/panel/me` | Current user identity |
| `GET` | `/panel/context-notes` | User context notes |
| `PUT` | `/panel/context-notes` | Save context notes |
| `GET` | `/swagger/doc.json` | OpenAPI spec (fetched by MCP `get_api_spec`) |
| `GET` | `/mcp/context` | Context doc listing → JSON array (fetched by MCP `list_context`) |
| `GET` | `/mcp/context/{path}` | One context doc → markdown (fetched by MCP `read_context`) |

The last two routes implement the **standard Open MCP context convention**. Every domain service exposes the same two paths so MCP can discover and fetch context docs without knowing where they are stored. Domain teams update their docs by redeploying their service — MCP needs no changes.

Regenerate Swagger after handler changes:

```powershell
swag init -g cmd/server/main.go -o docs --parseDependency --parseInternal
```

---

## Standard Open MCP service convention

Every Open MCP domain service must expose these two paths so MCP can work without configuration changes:

| Path | Returns | Purpose |
|------|---------|---------|
| `GET /swagger/doc.json` | OpenAPI JSON | API spec — fetched by `get_api_spec` |
| `GET /mcp/context` | JSON array of paths | Context listing — fetched by `list_context` |
| `GET /mcp/context/{path}` | Markdown/text | Context file — fetched by `read_context` |

The panel (`cmd/server`) implements all three. Infra and other domain services must do the same when registered.

| Tool | What it does |
|------|-------------|
| `list_sources` | List all registered domains with descriptions |
| `get_api_spec` | Fetch + cache OpenAPI/Swagger for a domain |
| `refresh_api_spec` | Force re-fetch Swagger |
| `call_api` | Proxy authenticated HTTP call to a domain API |
| `list_context` | List curated docs for a domain |
| `read_context` | Read one context markdown file |
| `get_user_context` | Current user identity + allowed projects + context notes |

---

## Project layout

DDD layout per `Open MCP-Documents/rules/10-go-backend.mdc` — one folder per domain
under `internal/domain/<name>/`, Go's `internal/` enforcing privacy.

```
cmd/server/          Panel HTTP server (:8080) — thin composition root
cmd/migrate/         Standalone schema + seed runner
cmd/mcp/             MCP HTTP server (:8081, Phase 2 — not yet created)

config/
  sources.yaml       Domain registry — platform_domain, swagger_path, context_path, auth env vars
  sources.yaml.example

context/
  project_id/        ID domain — source files served by panel at GET /mcp/context/*
  project_infra/     Infra domain — source files served by infra service at GET /mcp/context/*

internal/domain/id/  ID domain (IdP, panel auth, sessions, context notes)
  entity/            Domain entities + value objects (User, SessionClaims)
  repository/        Repository port interfaces
  service/           AuthService interface + business logic
  adapter/           Postgres (GORM) adapters, models, db connection, seed
  delivery/          HTTP handlers + self-registering routes
  wire.go            Domain composition (ProviderSet once google/wire lands)

internal/domain/mcp/ MCP platform domain (Phase 2 — not yet created)
internal/docs/       Swagger request/response models
docs/                Generated swagger spec (docs.go) + design documents + TODO

web/panel/           Login UI

Open MCP-Documents/    Authoritative platform docs (rules, domain specs)
```

---

## Seeded users

| Username | Password | Role |
|----------|----------|------|
| `admin` | `admin` | Admin — **change before going to production** |
| `service-bot` | `dev-service-token-change-me` | Service account — **change before going to production** |
