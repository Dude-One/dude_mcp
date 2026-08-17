# ID domain (Identity Panel)

Platform domain: **ID** — authentication, sessions, users, and per-user MCP context.

## When to use this domain

- Resolve **who is logged in** (user, tenant, allowed projects).
- Read or update **context notes** (markdown stored on the user profile).
- Panel session APIs — not device or asset data.

## Key concepts

| Term | Meaning |
|------|---------|
| `dude_mcp_session` | Opaque panel session cookie / Bearer token |
| `allowed_projects` | Platform domains or projects this user may access via MCP |
| `context_notes` | Free-form markdown injected into AI context for this user |

## Live API

Swagger is fetched dynamically by MCP from `/swagger/doc.json` on the panel service.
Do not duplicate route lists here — use `get_api_spec("dude-mcp-panel")` in the IDE.

## Related domains

- **Infra** (`dude-mcp-servermanagement`) — devices, assets, playbooks (separate Core service).
- **Device** — enrollment and telemetry (when registered in MCP).

## Authoritative docs

See `Dude MCP-Documents/ID/` for Auth, Users, MFA, and ACL specifications.
