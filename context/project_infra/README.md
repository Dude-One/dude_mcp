# Infra domain (Server Management)

Platform domain: **Infra** — servers, devices, assets, playbooks, and provisioning.

## When to use this domain

- Count or list **devices** (online/offline, by status or tag).
- Inspect **asset health** and automation status.
- Run read-only queries against server management APIs.

## Key concepts

| Term | Meaning |
|------|---------|
| Device status | Interpret via context guides — live values come from API |
| Asset | Managed endpoint under a tenant scope |
| Playbook | Automation workflow (writes require allowlisted tools) |

## Live API

Swagger is fetched dynamically by MCP from the server management service.
Use `get_api_spec("open-mcp-servermanagement")` — routes are not duplicated here.

## Related domains

- **ID** (`open-mcp-panel`) — tenant and user scope for MCP calls.
- **Device** — enrollment and telemetry may overlap; prefer the registered source in `list_sources`.

## Authoritative docs

See `Open MCP-Documents/` Infra and Server Management material when published.
