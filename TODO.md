# TODO — Dude MCP

## 1. Decouple auth from the built-in panel
The MCP server currently depends on the identity panel as its IdP.
Make it standalone: support per-source auth config (Bearer token, API key, basic auth) via `sources.yaml` so it works without a running panel or DB.

## 2. Decouple the panel from the MCP server
The admin UI + identity server is bundled with the MCP binary.
Split them so the MCP server runs as a single self-contained binary with no DB requirement for basic usage.

## 3. Flexible per-source auth in sources.yaml
Current source config has no auth fields.
Add support for: Bearer token, API key (header/query), Basic auth — all configurable per source, with env var interpolation for secrets.

## 4. Add tests
Zero test coverage makes it hard to refactor and harder to attract contributors.
Start with service-layer unit tests and at least one integration test for the MCP tool flow (list_sources → get_api_spec → call_api).
