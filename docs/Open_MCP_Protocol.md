# Open MCP — MCP Protocol Integration (Cursor / Claude / Codex)

> **Gate:** HTTP MCP server (`cmd/mcp`)  
> **Transport:** Streamable HTTP (`POST /mcp`) — deployed web service  
> **Design:** [`PROJECT_CONTEXT_AI_DESIGN.md`](./PROJECT_CONTEXT_AI_DESIGN.md)

---

## How it works

MCP is JSON-RPC 2.0 between the IDE's LLM and **your MCP web service**. MCP does not run the LLM.

1. IDE connects to `https://mcp.yourcompany.com/mcp` with a panel session token.
2. LLM calls **tools** (`list_sources`, `get_api_spec`, `call_api`, `read_context`, …).
3. MCP reads **`config/sources.yaml`** to find each domain's Swagger URL and context path.
4. MCP **fetches Swagger at runtime** — API routes stay current when domain services deploy.
5. MCP fetches **context docs** from `GET base_url + context_path + "/" + file` on each domain service.
6. LLM answers using live API data + curated docs.

```
User asks Cursor a question
  → LLM calls get_api_spec("open-mcp-servermanagement")
  → MCP fetches OpenAPI from registered swagger_path (cached)
  → LLM calls call_api(...) for live data
  → LLM optionally read_context(...) for domain guides
  → Answer shown in chat
```

---

## Domain registry

Domains are registered in `config/sources.yaml` — not hardcoded in MCP tools.

Each entry includes:

- `name` — tool argument (e.g. `open-mcp-panel`)
- `platform_domain` — official Open MCP domain key: `id`, `infra`, `device`, … matches `context/project_{platform_domain}/`
- `description` — **domain docstring** for the LLM (`list_sources`)
- `type` — `api` (Swagger + call_api) or `context` (docs only)
- `base_url` — service root URL; used for both Swagger and context fetches
- `swagger_path` — standard path: `GET {base_url}/swagger/doc.json`
- `context_path` — standard path: `GET {base_url}/mcp/context/{file}`

**Standard path convention** — all Open MCP services implement:

| Resource | Endpoint |
|----------|----------|
| OpenAPI spec | `GET /swagger/doc.json` |
| Context listing | `GET /mcp/context` → JSON array |
| Context file | `GET /mcp/context/{path}` → markdown/text |

See [`config/sources.yaml.example`](../config/sources.yaml.example).

---

## MCP tools (Phase 1)

| Tool | Purpose |
|------|---------|
| `list_sources` | Domains + descriptions from registry |
| `get_api_spec` | Fetch/cache OpenAPI for an API source |
| `refresh_api_spec` | Force Swagger re-fetch |
| `call_api` | Proxy authenticated HTTP to domain service |
| `list_context` | List docs from `GET {base_url}/mcp/context` on the domain's service |
| `read_context` | Fetch one file from `GET {base_url}/mcp/context/{path}` |
| `get_user_context` | Panel user identity + notes (session token) |

**Resources** (optional later): `trio://context/{domain}/doc/{path}`

---

## IDE configuration (production)

```json
{
  "mcpServers": {
    "trio": {
      "url": "https://mcp.yourcompany.com/mcp",
      "headers": {
        "Authorization": "Bearer <panel session token>"
      }
    }
  }
}
```

No `DATABASE_URL` or service API keys in the client config — those live on the MCP server.

---

## Legacy note

Earlier drafts used stdio transport and `ask_about_project` with a built-in LLM gateway. **Current direction:** HTTP MCP only; user's IDE runs the LLM; dynamic Swagger + context registry. See full design doc for details.
