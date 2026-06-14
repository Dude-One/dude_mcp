# Trio MCP — Repo-Native Integration (Cursor IDE)

> **Gate:** Direct file read — no server, no network  
> **Transport:** Filesystem  
> **Phase:** 1 — Zero cost, ships with MVP

---

## How It Works

Cursor IDE reads your repository's files as part of its context window.  
By placing `AGENTS.md` and `llms.txt` at the repo root, you tell Cursor exactly where your project documentation lives.  
Cursor then injects the relevant docs into its prompts **automatically** — without any API call or server.

```
Developer opens Cursor in the repo
    → Cursor reads AGENTS.md at repo root
    → Finds pointer to context/project_trio-servermanagement/
    → Injects relevant .md files into its context window
    → Developer asks a question → Cursor answers from your docs
```

This is the **cheapest and fastest** integration. No deployment, no auth, no latency.  
It only works for developers who have the repo checked out locally.

---

## Files to Create

### 1. `AGENTS.md` — repo root

Cursor reads this file to understand the project structure and where documentation lives.

```markdown
# Trio Project — Agent Context

## Project Overview
This repository contains the TrioV2 business logic service built with hexagonal (DDD)
architecture in Go. It manages server provisioning, playbooks, device management,
and project-scoped AI knowledge.

## Documentation Structure
All project documentation lives under `context/`:

- `context/project_trio-servermanagement/` — Server management domain docs
  - `README.md` — Project overview and quickstart
  - `architecture/` — System design, DDD mapping, data flow diagrams
  - `api/` — REST endpoint references and OpenAPI specs
  - `guides/` — Step-by-step operational guides
  - `examples/` — Playbook YAML samples and usage examples

## Key Concepts
- **Playbooks** — YAML-defined automation scripts for server operations
- **Devices** — Physical/virtual machines managed through the platform
- **Context AI** — Project-scoped knowledge assistant (this service)
- **Hexagonal architecture** — Domain core never imports infrastructure packages

## Coding Conventions
- Domain entities: `core/domain/entities/`
- Ports (interfaces): `core/ports/`
- Use cases: `core/usecases/`
- Adapters (implementations): `adapters/`
- REST handlers: `internal/rest/`
- Main wiring: `container.go`, `router.go`

## When answering questions about this repo:
1. Check `context/project_*/` for domain knowledge before guessing.
2. Ports define the contract — never bypass them.
3. LLM SDK calls belong only in `adapters/llm/`, never in `core/`.
```

---

### 2. `llms.txt` — repo root

`llms.txt` is a lightweight convention (similar to `robots.txt`) that tells LLMs what this repo/service is about and where to find documentation. Reduces token consumption by ~90% vs crawling the full repo.

```
# Trio Context AI Service — llms.txt

## What this service does
A project-scoped knowledge assistant. Stores curated documentation per project
under context/project_{Name}/ and exposes it to AI tools via REST API, MCP
protocol, and repo-native file context.

## Primary documentation
context/project_trio-servermanagement/README.md
context/project_trio-servermanagement/architecture/overview.md
context/project_trio-servermanagement/api/endpoints.md
context/project_trio-servermanagement/guides/

## OpenAPI spec
context/project_trio-servermanagement/api/openapi.yaml

## Source layout
core/domain/entities/   — domain entities
core/domain/services/   — DocumentSelector, PromptComposer
core/ports/             — LLMGateway, ContextDocumentRepository, ProjectContextAIService
core/usecases/          — Ask() orchestration
adapters/context/       — filesystem repository
adapters/llm/           — OpenAI, Ollama gateway implementations
adapters/mcp/           — MCP server adapter
internal/rest/          — HTTP handlers

## Do not read
.env
cmd/config/
adapters/repositories/  (contains credential handling)
```

---

## Context Directory Structure

Create and populate this tree. Cursor will read these files automatically.

```
context/
  project_trio-servermanagement/
    README.md                        ← copied/linked from repo root README
    architecture/
      overview.md                    ← DDD mapping, hexagonal wiring diagram
      data-flow.md                   ← Ask() pipeline step by step
      decisions.md                   ← ADRs (architecture decision records)
    api/
      endpoints.md                   ← REST endpoint reference
      openapi.yaml                   ← OpenAPI 3.1 spec (same as Trio_MCP_OpenAI.md)
    guides/
      quickstart.md                  ← getting started
      playbooks.md                   ← how to write and run playbooks
      provisioning.md                ← server provisioning steps
    examples/
      create-server.yaml             ← example playbook
      assign-device.yaml             ← example playbook
```

---

## Seeding Content

Populate the context tree by copying existing repo docs:

```powershell
# Create the directory tree
New-Item -ItemType Directory -Path "context\project_trio-servermanagement\architecture" -Force
New-Item -ItemType Directory -Path "context\project_trio-servermanagement\api" -Force
New-Item -ItemType Directory -Path "context\project_trio-servermanagement\guides" -Force
New-Item -ItemType Directory -Path "context\project_trio-servermanagement\examples" -Force

# Copy root README as the project overview
Copy-Item README.md "context\project_trio-servermanagement\README.md"

# Copy existing docs if they exist
Copy-Item docs\* "context\project_trio-servermanagement\guides\" -Recurse -ErrorAction SilentlyContinue
Copy-Item examples\*.yaml "context\project_trio-servermanagement\examples\" -ErrorAction SilentlyContinue
```

---

## Markdown Frontmatter (Optional but Recommended)

Adding YAML frontmatter to context docs helps the `DocumentSelector` score them more accurately:

```markdown
---
title: Server Provisioning Guide
tags: [provisioning, server, operations]
---

# Server Provisioning Guide
...
```

Supported fields parsed by `FSContextRepository`:

| Field | Type | Used for |
|-------|------|---------|
| `title` | string | Display name in sources |
| `tags` | string[] | Keyword scoring in DocumentSelector |
| `description` | string | Summary for search |

---

## Constraints

| Constraint | Value |
|------------|-------|
| Max file size | 512 KB per file |
| Max project total | Configurable via `CONTEXT_ROOT_PATH` |
| Supported formats | `.md`, `.yaml`, `.yml`, `.txt`, `.json` |
| Skipped paths | hidden dirs, `node_modules`, binaries |
| Project dir naming | Must match `project_{Name}` — name: `^[a-zA-Z0-9_-]+$` |

---

## How Cursor Uses These Files

1. Cursor reads `AGENTS.md` on startup and keeps it in context.
2. When you ask a question, Cursor checks `llms.txt` to understand the service.
3. If Cursor needs more detail, it reads individual files from `context/project_*/`.
4. No server call is made — everything is local filesystem reads.
5. Cursor's context window limits apply (~200k tokens for Claude-based models).

For large doc trees (> ~100k tokens), switch to the **MCP gate** (Phase 2) or **REST gate** which do server-side document selection instead of injecting everything.

---

## What NOT to Do

- Do not put `.env` values, passwords, or API keys in any file under `context/`.
- Do not symlink `context/` to `core/` or `adapters/` — docs only.
- Do not write `AGENTS.md` as a pure tech spec — write it so an AI can understand intent and structure.
- Do not skip frontmatter on docs you want ranked accurately by the document selector.
