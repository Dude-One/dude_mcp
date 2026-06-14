# Open MCP — TODO

> Implementation order from Phase 0 → production.
> DDD layout: `internal/domain/<name>/{entity,repository,service,adapter,delivery,wire.go}`

---

## Status legend

| Symbol | Meaning |
|--------|---------|
| `[ ]`  | Not started |
| `[~]`  | In progress |
| `[x]`  | Done |

---

## Phase 1 — Identity panel ✅ DONE

- [x] Panel auth — login, signup, logout, session cookie, context notes
- [x] Postgres — tenants, users, user_sessions (schema: `open_mcp_db`)
- [x] Swagger at `/swagger/doc.json`
- [x] Seed context trees — `context/project_id/`, `context/project_infra/`
- [x] DDD layout migration — `internal/domain/id/{entity,repository,service,adapter,delivery,wire.go}`
- [x] Health endpoints — `GET /health/live`, `GET /health/ready`
- [x] Self-registering routes — `delivery/routes.go`

---

## Phase 2 — MCP platform ✅ DONE

- [x] `config/sources.yaml` — open-mcp-panel + open-mcp-agentless registered
- [x] `.env` — MCP_PORT, SWAGGER_CACHE_TTL, PANEL_USERNAME/PASSWORD, AGENTLESS_COMPANY_ID
- [x] `internal/domain/mcp/{entity,repository,service,adapter,delivery,wire.go}`
- [x] 5 MCP tools: list_sources, get_api_spec, list_context_docs, read_context_doc, call_api
- [x] Auth: password (panel) + query_param (agentless) strategies
- [x] `cmd/mcp/main.go` — HTTP Streamable JSON-RPC on :8081
- [x] `.cursor/mcp.json` + `.vscode/launch.json`
- [x] Tested: Cursor + PowerShell curl — both sources live, real data confirmed

---

## Phase 3 — Write operations + full engine test ✅ DONE

- [x] All HTTP methods allowed (GET, HEAD, POST, PUT, PATCH, DELETE)
- [x] Access control delegated to ID domain token layer (Phase 5) — not MCP allowlist
- [x] Request body support for POST/PUT/PATCH (Content-Type: application/json)
- [x] Path traversal guard kept (`..` rejected)
- [x] Unrecognised method guard kept

---

## Phase 4 — Second domain + ACL

- [ ] Register additional domains in `sources.yaml` when their APIs + Swagger exist
- [ ] `allowed_projects` ACL — filter sources by user's `allowed_projects` list
- [ ] Audit log — persist `user_id`, `source`, `method`, `path` on every `call_api`
- [ ] Add `context/project_infra/guides/` — device status semantics, playbook notes
- [ ] `GET /health/ready` in `cmd/mcp/main.go` — panel reachability check

---

## Phase 5 — MCP access tokens (auth upgrade from MVP)

> Replaces username+password in config. No other Phase 2 code changes.

**Why:** password in config is plaintext, breaks on MFA, breaks on password rotation.
**What changes:** only the middleware auth method. Everything else stays the same.

- [ ] `id` domain — `entity/mcp_token.go` — `MCPToken` (token, user_id, tenant_id, scopes, expires_at)
- [ ] `id` domain — `adapter/mcp_token_repository.go` — create, find, revoke in DB
- [ ] `id` domain — `service/` — `IssueToken()`, `ValidateToken()`, `RevokeToken()`
- [ ] `id` domain — `delivery/` — add 3 endpoints:
  - `POST   /panel/mcp-tokens` — generate a new token
  - `GET    /panel/mcp-tokens` — list my active tokens
  - `DELETE /panel/mcp-tokens/:id` — revoke a token
- [ ] `mcp` domain — `delivery/middleware.go` — swap credential resolution:
  - was: read username+password → login → session
  - now: read `Bearer mcp_xxx` → call `id.TokenValidator` → get user+tenant
- [ ] `.cursor/mcp.json` — swap `username+password` for `Bearer mcp_xxx`
- [ ] Panel UI — add "MCP Tokens" tab (generate / list / revoke)

---

## Removed / deprioritized

- ~~OAuth / ChatGPT Custom GPT~~ — historical, see `Trio_Auth_OAuth_Design.md`
- ~~REST `/context-ai/ask` with OpenAI~~ — MCP + IDE LLM is the primary path
- ~~stdio MCP transport~~ — HTTP only (shared service, multi-user)
- ~~HTTP SSE transport~~ — deprecated MCP spec; using HTTP Streamable

---

## Design docs index

| Doc | Covers |
|-----|--------|
| [`Phase2_MCP_Plan.md`](./Phase2_MCP_Plan.md) | Review, decisions, resequenced build order |
| [`PROJECT_CONTEXT_AI_DESIGN.md`](./PROJECT_CONTEXT_AI_DESIGN.md) | Full MCP design — registry, domain alignment, tools |
| [`../Open MCP-Documents/Platform/MCP.md`](../Open MCP-Documents/Platform/MCP.md) | Platform-level MCP doc |
| [`Trio_MCP_Protocol.md`](./Trio_MCP_Protocol.md) | IDE integration — transport, tools, config |
| [`Open MCP Server for IT Operations Management.md`](./Trio%20MCP%20Server%20for%20IT%20Operations%20Management.md) | End-state ITOM vision |
