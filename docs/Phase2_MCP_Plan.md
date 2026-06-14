# Phase 2 — MCP Platform: Review, Decisions, and Resequenced Plan

> Handoff doc for implementation (Claude Code or otherwise).
> Captures a design review of the current Phase 2 plan, the open decisions to settle
> before coding, and a build order with security work pulled forward.
>
> Source plan: [`TODO.md`](./TODO.md) Phase 2 · [`PROJECT_CONTEXT_AI_DESIGN.md`](./PROJECT_CONTEXT_AI_DESIGN.md) · [`../Open MCP-Documents/Platform/MCP.md`](../Open MCP-Documents/Platform/MCP.md)
> Governance: [`../Open MCP-Documents/rules/00-trio-core.mdc`](../Open MCP-Documents/rules/00-trio-core.mdc) and the other `rules/*.mdc`.

---

## Status snapshot (verified against code)

- **Phase 1 (Identity panel) — done.** `core/auth/`, `adapters/auth/persistence/`, `internal/app/`, `internal/auth/panel_handler.go`, `cmd/migrate`. Panel serves `/mcp/context` via `internal/mcp/context_handler.go` from `context/project_id/`.
- **Phase 2 (MCP platform) — not started.** Confirmed missing: `config/sources.yaml` (only `.example`), `cmd/mcp/`, `core/mcp/` (or `internal/domain/mcp/`), `adapters/mcp/`, `internal/mcp/server.go|handlers.go|middleware.go`, and the `mark3labs/mcp-go` dependency.

The bones of the plan are right: registry-driven routing, runtime Swagger discovery, "add a domain = a YAML entry, no code change." The phasing is sensible. The weak part is the **trust / credential model**, and it is mostly deferred to later phases when it should be settled now.

---

## Review findings

### 1. IDE→MCP credential is the wrong type (auth)

"Paste your panel session token into Cursor config" puts a browser session token into a plaintext config file on a developer laptop. Sessions expire, revoke on logout, and rotate — so the integration breaks unpredictably while the credential sits durably on disk. Session tokens authenticate a browser; they should not authorize a long-running tool integration.

**Recommendation:** dedicated **MCP access tokens** (PAT-style) with their own lifecycle, scopes, revocation, and audit, decoupled from panel sessions. This is the gap OAuth device flow normally fills — deprioritizing OAuth was fine, but it left token handling as the softest spot in the design.

### 2. `call_api` + shared downstream service token is a multi-tenant hazard (security)

MCP authenticates to Infra/Device with one privileged server-side token (e.g. `SM_API_KEY`) regardless of which user is asking, then "passes tenant scope as a header per target Swagger." That inverts the non-negotiable rule — *every query scoped by `tenant_id`, enforced at the repository layer, tested explicitly*. Here isolation depends on MCP always injecting the right scope **and** every downstream enforcing it against a god token. One forgetful downstream = cross-tenant leak. A generic authenticated proxy over arbitrary Swagger paths is also an SSRF/IDOR amplifier.

The plan's mitigations (GET/HEAD only, reject `..`, audit) are good but the **per-source path allowlist and MCP-layer tenant enforcement sit in Phase 4**. Pull them into Phase 2 — the first `call_api` implementation. Do not ship a generic proxy to live infra APIs without them.

### 3. Build MCP in the target layout now, not transitional (architecture)

`core/auth` + `adapters/auth` is transitional because it predates the rule — earned debt. MCP is brand-new in Phase 2 and has no such excuse. Building in `core/mcp` guarantees a future migration.

**Recommendation:** build directly in `internal/domain/mcp/{entity,repository,service,delivery,adapter,middleware,wire.go}` from the first commit, per `10-go-backend.mdc` and the design doc's target layout.

### 4. Context docs have three potential sources of truth (docs / drift)

`Open MCP-Documents/<Domain>/` is "authoritative," `context/project_*/` are "IDE-oriented summaries," and Swagger descriptions are a third. Nothing keeps them in sync, so the LLM will eventually read stale context and act on it confidently.

**Recommendation:** declare what is canonical and how summaries are derived/validated. Also resolve the stated-vs-actual point: the design says context docs are served by each domain service and *not* stored on MCP disk, but Phase 1 stores them on disk in `context/project_id/`. That is fine under the panel-is-the-ID-service reading — write it down so it is not read as a contradiction.

### 5. Swagger cache vs. stateless-replica rule (scaling)

An in-process TTL cache is process-local state, which the stateless-replica rule treats with suspicion. It is a soft cache, so per-replica divergence is acceptable — but say so explicitly, or it trips the CI arch-linter / review checklist. If cross-replica freshness consistency is wanted, it becomes a Redis cache; otherwise document that each replica caches independently.

---

## Open decisions to settle before coding

| # | Decision | Options | Recommendation |
|---|----------|---------|----------------|
| D1 | IDE→MCP credential | Panel session token (plan) vs. dedicated MCP access token | **MCP access token** (own lifecycle/scopes/revocation). Session token acceptable only for a throwaway dev spike. |
| D2 | `call_api` safety scope for Phase 2 | Defer allowlist+tenant enforcement to Phase 4 (plan) vs. include now | **Include now**: per-source path allowlist + MCP-layer tenant scope enforcement + GET/HEAD only. |
| D3 | Code layout | Transitional `core/mcp` vs. target `internal/domain/mcp/` | **Target layout** from first commit. |
| D4 | Context-doc canonical source | Undeclared (3 candidates) | Declare `Open MCP-Documents/<Domain>/` authoritative; `context/project_*/` are derived summaries. |
| D5 | Swagger cache | In-process TTL vs. Redis | In-process TTL is fine if documented as soft/per-replica; move to Redis only if freshness consistency is required. |

---

## Resequenced Phase 2 build order

Security items (originally Phase 4) are folded into the first `call_api`. Build in the **target** `internal/domain/mcp/` layout.

### Step 1 — Registry + config
- [ ] `config/sources.yaml` — copy from example; register `open-mcp-panel` (`platform_domain: id`, `swagger_path: /swagger/doc.json`, `context_path: /mcp/context`). **Add a per-source `allowlist` of permitted method+path patterns** (D2).
- [ ] `.env` (single file, inline comments) — `MCP_PORT`, `PANEL_SERVICE_TOKEN`, `SWAGGER_CACHE_TTL`, and the MCP-access-token settings from D1.

### Step 2 — Core domain (`internal/domain/mcp/`)
- [ ] `entity/source.go` — `Source`, `CachedSpec`; include `Allowlist` on `Source`.
- [ ] `repository/ports.go` — `SourceRegistry`, `SwaggerReader`, `ContextReader`, `APICaller` interfaces.
- [ ] `service/mcp_service.go` — `MCPServiceImpl` (tool orchestration). **Tenant scope + allowlist enforced here, before any downstream call** (D2).

### Step 3 — Adapters (`internal/domain/mcp/adapter/`)
- [ ] `yaml_registry.go` — load `sources.yaml`, serve `SourceRegistry`.
- [ ] `http_swagger_reader.go` — fetch + cache Swagger (TTL from `SWAGGER_CACHE_TTL`). Document cache as soft/per-replica (D5).
- [ ] `http_api_caller.go` — proxy `call_api`. **GET/HEAD only; deny non-allowlisted paths; inject tenant scope; reject `..`.**
- [ ] `http_context_reader.go` / `fs_context_reader.go` — read context per the canonical-source decision (D4).

### Step 4 — Delivery (`internal/domain/mcp/delivery/`)
- [ ] `server.go` — register MCP tools with `mark3labs/mcp-go`.
- [ ] `handlers.go` — thin tool handlers; delegate to service.
- [ ] `middleware.go` — **MCP access-token auth** (D1), audit log on every `call_api` (`user_id`, `source`, `method`, `path`).

### Step 5 — Entry point + wiring
- [ ] `internal/domain/mcp/wire.go` — Wire `ProviderSet`, self-registering routes.
- [ ] `cmd/mcp/main.go` — HTTP Streamable JSON-RPC on `POST /mcp`, listen on `MCP_PORT`.
- [ ] `go.mod` — add `github.com/mark3labs/mcp-go`.

### Step 6 — IDE wiring
- [ ] `.cursor/mcp.json` (dev) — point to `:8081`, use MCP access token (not session token).
- [ ] `.vscode/launch.json` — add "Open MCP: MCP Server" debug config.

### Step 7 — Verify (acceptance, not just code paths)
- [ ] `list_sources` returns `open-mcp-panel`.
- [ ] `get_api_spec("open-mcp-panel")` returns live Swagger from panel.
- [ ] `get_user_context` returns current user via MCP access token.
- [ ] `read_context("id", "README.md")` returns `context/project_id/README.md`.
- [ ] **`call_api` to a non-allowlisted path is denied.**
- [ ] **`call_api` cannot reach another tenant's data** (multi-tenant boundary test).
- [ ] **Non-GET/HEAD `call_api` is rejected in Phase 2.**

---

## Change-impact sides this touches (from `00-trio-core.mdc`)

Expect to update: **Code** (new bounded context), **DB** (MCP access tokens table if D1 adopted — its own schema), **Documents** (this doc + `Platform/MCP.md` changelog), **Tests** (allowlist, tenant-boundary, transport), **ACL** (`allowed_projects` filtering — at least stubbed), **Activity Log** (audit on `call_api`), **API contract** (MCP tool contract), **ENV** (single `.env`), **Security** (the whole point of D2), **On-prem + air-gap** (no runtime outbound internet; registry URLs internal).

---

## Assumptions & open questions

- **Assumed:** the panel acts as the ID domain service for the `/mcp/context` convention; storing `context/project_id/` on the panel disk is the intended Phase-1 form, not a contradiction of the "domains serve their own docs" rule.
- **Assumed:** `mark3labs/mcp-go` is the chosen MCP library (named in `TODO.md`). Confirm before adding the dependency.
- **Open (D1):** is there appetite to add an MCP-access-token table + minimal issue/revoke flow in Phase 2, or is the session-token shortcut acceptable for the first internal milestone? This is the single highest-impact decision.
- **Open (D4):** who owns keeping `context/project_*/` summaries in sync with `Open MCP-Documents/<Domain>/`? Manual, or generated?
- **Open:** there is no root `CLAUDE.md` in the repo today (only `Open MCP-Documents/rules/*.mdc`). If Claude Code should auto-load governance, add a root `CLAUDE.md` that references the rules folder.
