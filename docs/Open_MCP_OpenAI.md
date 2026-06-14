# Open MCP — OpenAI / ChatGPT Integration

> **Gate:** REST API → ChatGPT Custom GPT Actions  
> **Transport:** HTTPS (plain HTTP POST/GET)  
> **Phase:** 1 — MVP

---

## How It Works

ChatGPT does **not** call your server through a library or SDK.  
You write one **OpenAPI 3.1 YAML spec** that describes your endpoints.  
You paste it into the ChatGPT Custom GPT "Actions" panel.  
ChatGPT reads the spec, decides which endpoint to call based on the user's question, and fires an HTTP request — exactly like any HTTP client would.

```
User question (natural language)
    → ChatGPT reads OpenAPI spec
    → Picks the right operationId
    → POST /context-ai/projects/:name/ask  { "question": "..." }
    → Your server runs Ask() pipeline
    → Returns JSON answer + sources
    → ChatGPT rephrases in natural language → User
```

---

## REST Endpoints to Implement

| Method | Path | Purpose |
|--------|------|---------|
| `GET`  | `/context-ai/projects` | List all available project names |
| `GET`  | `/context-ai/projects/:name/documents` | List document tree metadata for a project |
| `POST` | `/context-ai/projects/:name/ask` | Ask a question — runs full Ask() pipeline |
| `POST` | `/context-ai/projects/:name/refresh` | Force re-scan after doc updates |

---

## OpenAPI 3.1 Spec

Paste this into the ChatGPT Custom GPT → **Configure** → **Actions** → **Schema** box.

```yaml
openapi: 3.1.0
info:
  title: Trio Context AI
  description: >
    Knowledge assistant for Open MCP projects. Ask questions about project
    documentation, architecture, APIs, and playbooks. Returns answers
    with cited source documents.
  version: 1.0.0

servers:
  - url: https://your-server.com
    description: Production

paths:

  /context-ai/projects:
    get:
      operationId: listProjects
      summary: List available projects
      description: >
        Returns all project names that have a context document tree.
        Use this to discover which project name to pass to other endpoints.
      responses:
        "200":
          description: List of project names
          content:
            application/json:
              schema:
                type: object
                properties:
                  projects:
                    type: array
                    items:
                      type: string
                    example: ["open-mcp-servermanagement", "trio-billing"]

  /context-ai/projects/{name}/documents:
    get:
      operationId: listDocuments
      summary: List documents in a project
      description: >
        Returns metadata (path, title, tags, kind) for all documents in the
        project's context tree. Does not return full content — use this to
        understand what documentation is available before asking a question.
      parameters:
        - name: name
          in: path
          required: true
          description: Project name (e.g. "open-mcp-servermanagement")
          schema:
            type: string
      responses:
        "200":
          description: Document metadata list
          content:
            application/json:
              schema:
                type: object
                properties:
                  project:
                    type: string
                  documents:
                    type: array
                    items:
                      $ref: "#/components/schemas/DocumentMeta"
        "404":
          description: Project not found

  /context-ai/projects/{name}/ask:
    post:
      operationId: askAboutProject
      summary: Ask a question about a project
      description: >
        Runs the full knowledge pipeline: loads the project's document tree,
        selects the most relevant documents, composes a prompt, calls the LLM,
        and returns a natural language answer with cited source documents.
        Use this as the primary endpoint for answering user questions.
      parameters:
        - name: name
          in: path
          required: true
          description: Project name (e.g. "open-mcp-servermanagement")
          schema:
            type: string
      requestBody:
        required: true
        content:
          application/json:
            schema:
              type: object
              required:
                - question
              properties:
                question:
                  type: string
                  description: The user's question in natural language
                  example: "How do I create a new server in open-mcp-servermanagement?"
                max_sources:
                  type: integer
                  description: Max number of source documents to include (default 5)
                  default: 5
                  minimum: 1
                  maximum: 20
      responses:
        "200":
          description: Answer with cited sources
          content:
            application/json:
              schema:
                $ref: "#/components/schemas/KnowledgeAnswer"
        "404":
          description: Project not found
        "429":
          description: Rate limited — one concurrent Ask per project
        "504":
          description: LLM timeout

  /context-ai/projects/{name}/refresh:
    post:
      operationId: refreshProject
      summary: Refresh a project's document cache
      description: >
        Forces a re-scan of the project's context directory on disk. Call this
        after updating documentation files to ensure the next Ask() uses
        the latest content.
      parameters:
        - name: name
          in: path
          required: true
          schema:
            type: string
      responses:
        "200":
          description: Refresh complete
          content:
            application/json:
              schema:
                type: object
                properties:
                  project:
                    type: string
                  documents_indexed:
                    type: integer
        "404":
          description: Project not found

components:
  schemas:

    DocumentMeta:
      type: object
      properties:
        path:
          type: string
          description: Relative path within the project tree
          example: "guides/playbooks.md"
        title:
          type: string
          description: Document title from frontmatter or filename
          example: "Playbooks Guide"
        kind:
          type: string
          enum: [markdown, yaml, text, json]
        tags:
          type: array
          items:
            type: string
          example: ["playbook", "operations"]

    DocumentRef:
      type: object
      properties:
        path:
          type: string
          description: Source document path (relative to project root)
        title:
          type: string

    KnowledgeAnswer:
      type: object
      properties:
        answer:
          type: string
          description: Natural language answer generated by the LLM
        sources:
          type: array
          description: Documents cited to produce the answer
          items:
            $ref: "#/components/schemas/DocumentRef"
        model:
          type: string
          description: LLM model used
          example: "gpt-4o"

  securitySchemes:
    ApiKeyAuth:
      type: apiKey
      in: header
      name: X-API-Key

security:
  - ApiKeyAuth: []
```

---

## Authentication

In the ChatGPT Custom GPT Actions panel set:

| Field | Value |
|-------|-------|
| **Auth type** | API Key |
| **Header name** | `X-API-Key` |
| **Key** | Your server's API key |

On your server, validate the `X-API-Key` header on every request.

---

## GPT Instructions (paste into Custom GPT "Instructions" box)

```
You are a Trio project assistant. You answer questions about Trio project
documentation — architecture, APIs, guides, and playbooks.

When a user asks a question:
1. Call listProjects if you don't know which project they mean.
2. Call askAboutProject with the correct project name and the user's question.
3. Present the answer clearly. Always cite the source documents returned.
4. If no relevant answer is found, say so honestly.

Never make up information not present in the returned answer.
Always include the "sources" paths so the user knows where the information came from.
```

---

## Limits & Constraints

| Constraint | Value |
|------------|-------|
| Request timeout | 45 seconds (ChatGPT hard limit) |
| Max payload | 100,000 characters |
| Concurrent Ask per project | 1 (rate limited by server) |
| TLS required | Yes — TLS 1.2+ on port 443 |
| Context docs exposed | `context/project_{Name}/` only — never engine source |

---

## Example Request / Response

**Request:**
```http
POST /context-ai/projects/open-mcp-servermanagement/ask
X-API-Key: sk-xxx
Content-Type: application/json

{
  "question": "What are the steps to provision a new server?",
  "max_sources": 3
}
```

**Response:**
```json
{
  "answer": "To provision a new server in open-mcp-servermanagement, follow these steps: ...",
  "sources": [
    { "path": "guides/provisioning.md", "title": "Server Provisioning Guide" },
    { "path": "examples/create-server.yaml", "title": "Create Server Playbook" }
  ],
  "model": "gpt-4o"
}
```

---

## Go Handler Skeleton — `internal/rest/project_context_ai_handler.go`

```go
type ProjectContextAIHandler struct {
    service ports.ProjectContextAIService
}

// GET /context-ai/projects
func (h *ProjectContextAIHandler) ListProjects(c *gin.Context) {
    projects, err := h.service.ListProjects(c.Request.Context())
    if err != nil {
        c.JSON(500, gin.H{"error": err.Error()})
        return
    }
    c.JSON(200, gin.H{"projects": projects})
}

// GET /context-ai/projects/:name/documents
func (h *ProjectContextAIHandler) ListDocuments(c *gin.Context) {
    name := c.Param("name")
    docs, err := h.service.ListDocuments(c.Request.Context(), name)
    if err != nil {
        c.JSON(404, gin.H{"error": "project not found"})
        return
    }
    c.JSON(200, gin.H{"project": name, "documents": docs})
}

// POST /context-ai/projects/:name/ask
func (h *ProjectContextAIHandler) Ask(c *gin.Context) {
    name := c.Param("name")
    var body struct {
        Question   string `json:"question"`
        MaxSources int    `json:"max_sources"`
    }
    if err := c.ShouldBindJSON(&body); err != nil {
        c.JSON(400, gin.H{"error": "invalid request"})
        return
    }
    if body.MaxSources == 0 {
        body.MaxSources = 5
    }
    answer, err := h.service.Ask(c.Request.Context(), entities.KnowledgeQuery{
        ProjectName: name,
        Question:    body.Question,
        MaxSources:  body.MaxSources,
    })
    if err != nil {
        c.JSON(500, gin.H{"error": err.Error()})
        return
    }
    c.JSON(200, answer)
}

// POST /context-ai/projects/:name/refresh
func (h *ProjectContextAIHandler) Refresh(c *gin.Context) {
    name := c.Param("name")
    if err := h.service.RefreshProject(c.Request.Context(), name); err != nil {
        c.JSON(500, gin.H{"error": err.Error()})
        return
    }
    c.JSON(200, gin.H{"project": name, "status": "refreshed"})
}
```

---

## What NOT to Do

- Do not expose `/context-ai/admin` or internal routes in the OpenAPI spec given to ChatGPT.
- Do not include source code paths, `.env` values, or database internals in context docs.
- Do not skip TLS — ChatGPT will refuse to call plain HTTP endpoints.
- Do not store the API key in the context docs tree.
