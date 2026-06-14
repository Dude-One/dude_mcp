# Open MCP Server for IT Operations Management
## 1. Goal
Build a reusable MCP Server for Trio that allows external LLM clients such as Claude, Cursor, and other MCP-compatible agents to securely access Trio product capabilities.
The first implementation target is IT Operations Management, but the architecture must support other Open MCP domains such as Command Center, Asset Management, Automation, and Reporting.
## 2. Core Principle
MCP should be implemented as a platform-level integration layer, not as a feature tightly coupled to IT Operations Management.
```text
LLM Client
  -> Open MCP Server
  -> Tool Registry
  -> Domain Tools
  -> Existing Trio Backend APIs

This keeps the system simple, reusable, and scalable.

⸻

3. High-Level Architecture

Claude / Cursor / MCP Client
        |
        | MCP Protocol
        v
+-----------------------+
| Open MCP Server       |
+-----------------------+
| Auth Context Resolver |
| Tool Registry         |
| Permission Guard      |
| Audit Logger          |
| Domain Adapters       |
+-----------------------+
        |
        v
+-----------------------+
| Trio Backend APIs     |
+-----------------------+
| ITOM                  |
| Command Center        |
| Assets                |
| Automation            |
| Reporting             |
+-----------------------+

⸻

4. Main Components

4.1 MCP Server

Responsible for:

* Exposing MCP tools to external clients
* Handling MCP transport
* Resolving user, tenant, and workspace context
* Validating permissions
* Routing tool calls to domain adapters
* Logging all tool executions

Initial transport:

Streamable HTTP (deployed web service at `POST /mcp`)

Optional later:

stdio (local dev only)

⸻

4.2 Tool Registry

The Tool Registry manages all tools exposed through MCP.

Each tool must have:

type MCPToolDefinition = {
  name: string;
  domain: string;
  description: string;
  inputSchema: object;
  outputSchema?: object;
  requiredPermissions: string[];
  riskLevel: "low" | "medium" | "high";
  requiresConfirmation: boolean;
};

Example:

{
  name: "itom.incident.search",
  domain: "itom",
  description: "Search IT operation incidents",
  requiredPermissions: ["incident:read"],
  riskLevel: "low",
  requiresConfirmation: false
}

⸻

4.3 Auth Context Resolver

Every tool call must run with a resolved execution context.

type MCPExecutionContext = {
  tenantId: string;
  workspaceId: string;
  userId: string;
  roles: string[];
  permissions: string[];
  clientName: string;
  requestId: string;
};

The MCP Server must never execute a tool without this context.

⸻

4.4 Permission Guard

Before executing any tool:

1. Resolve user context
2. Read tool permission requirements
3. Check user permissions
4. Block unauthorized access
5. Log the denied request

Example:

itom.incident.search        -> incident:read
itom.incident.create        -> incident:create
itom.incident.update_status -> incident:update
automation.playbook.run     -> automation:execute

⸻

4.5 Audit Logger

Every MCP tool call must be logged.

Minimum fields:

type MCPAuditLog = {
  requestId: string;
  tenantId: string;
  workspaceId: string;
  userId: string;
  clientName: string;
  toolName: string;
  input: object;
  status: "success" | "failed" | "denied";
  error?: string;
  createdAt: string;
};

⸻

5. Domain Adapter Pattern

MCP tools should not directly access databases.

Correct flow:

MCP Tool
  -> Domain Adapter
  -> Existing Trio Backend API

Example:

class ITOMAdapter {
  async searchIncidents(input, context) {
    return trioApiClient.get("/itom/incidents", {
      params: input,
      headers: buildContextHeaders(context)
    });
  }
}

This keeps MCP isolated from internal implementation details.

⸻

6. Initial ITOM Tools

Phase 1: Read-Only Tools

Implement these first:

itom.incident.search
itom.incident.get
itom.asset.search
itom.asset.get_health
itom.alert.list_open
itom.service.get_status

These tools are low risk and useful for Claude and Cursor immediately.

⸻

Phase 2: Write Tools

Add after read-only tools are stable:

itom.incident.create
itom.incident.add_comment
itom.incident.assign
itom.incident.update_status

Requirements:

* Permission check
* Audit log
* Input validation
* Idempotency key
* Optional user confirmation

⸻

Phase 3: Action Tools

Add later:

itom.diagnostic.run
itom.remediation.run_playbook
itom.incident.escalate

These are higher risk and must require confirmation.

⸻

7. Command Center Integration

Command Center should use the same MCP Server and Tool Registry.

Example tools:

command_center.overview.get
command_center.alerts.summarize
command_center.events.search
command_center.action.create
command_center.action.execute

Command Center tools can aggregate data from multiple domains.

Example:

command_center.overview.get
  -> incidents
  -> alerts
  -> asset health
  -> automation status

⸻

8. Recommended Project Structure

services/
  open-mcp-server/
    src/
      index.ts
      mcp/
        server.ts
        transport.ts
      auth/
        context-resolver.ts
      registry/
        tool-registry.ts
      policy/
        permission-guard.ts
      audit/
        audit-logger.ts
      adapters/
        trio-api-client.ts
        itom-adapter.ts
        command-center-adapter.ts
      tools/
        itom/
          incident.tools.ts
          asset.tools.ts
          alert.tools.ts
        command-center/
          overview.tools.ts
      schemas/
        itom.schema.ts

⸻

9. Tool Execution Flow

1. MCP client calls a tool
2. MCP Server receives request
3. Auth Context Resolver resolves user and tenant
4. Tool Registry finds tool definition
5. Permission Guard validates access
6. Input schema is validated
7. Domain Adapter calls Trio Backend API
8. Result is normalized
9. Audit log is written
10. MCP response is returned

⸻

10. Example Tool Implementation

export const searchIncidentsTool = {
  name: "itom.incident.search",
  description: "Search IT operation incidents",
  inputSchema: {
    type: "object",
    properties: {
      query: { type: "string" },
      status: { type: "string" },
      severity: { type: "string" },
      limit: { type: "number", default: 20 }
    }
  },
  requiredPermissions: ["incident:read"],
  riskLevel: "low",
  requiresConfirmation: false,
  async execute(input, context, adapters) {
    return adapters.itom.searchIncidents(input, context);
  }
};

⸻

11. Security Rules

The MCP Server must follow these rules:

* Never expose internal APIs directly
* Never execute a tool without user context
* Never bypass permission checks
* Never allow destructive actions without confirmation
* Always validate input schemas
* Always write audit logs
* Limit output size to avoid leaking excessive data
* Use allowlists for enabled tools per client

⸻

12. Client Configuration

Each MCP client can have its own allowed tool set.

{
  clientName: "cursor",
  enabledDomains: ["itom", "command_center"],
  allowedTools: [
    "itom.incident.search",
    "itom.incident.get",
    "itom.asset.get_health",
    "command_center.overview.get"
  ],
  blockedTools: [
    "automation.playbook.run"
  ]
}

⸻

13. MVP Scope

The MVP should include:

* MCP stdio transport
* Auth context resolver
* Tool registry
* Permission guard
* Audit logger
* ITOM read-only tools
* Basic Claude and Cursor configuration
* Documentation for adding new tools

Out of scope for MVP:

* Hosted MCP Gateway
* HTTP SSE transport (deprecated — using HTTP Streamable instead)
* Automation execution
* Destructive actions
* Cross-tenant tool discovery
* Advanced prompt registry

⸻

14. Implementation Phases

Phase 1: Foundation

* Create open-mcp-server
* Add MCP server runtime
* Add stdio transport
* Add tool registry
* Add context resolver
* Add audit logger

Phase 2: ITOM Read Tools

* Add ITOM adapter
* Add incident search
* Add incident details
* Add asset search
* Add asset health
* Add alert list

Phase 3: Write Support

* Add incident creation
* Add comments
* Add status update
* Add permission and confirmation rules
* Add idempotency support

Phase 4: Platform Expansion

* Add Command Center tools
* Add Automation tools
* Add Asset tools
* Add Reporting tools

Phase 5: Hosted MCP

* HTTP Streamable transport already in Phase 1 (production path)
* Add OAuth per-user auth (beyond Bearer panel token)
* Add tenant-aware remote MCP access
* Add admin configuration UI

⸻

15. Definition of Done

The implementation is complete when:

* Claude or Cursor can connect to Open MCP Server
* A user can search ITOM incidents through MCP
* Tool calls are permission-checked
* Tool calls are audited
* The MCP Server uses existing Open MCP backend APIs
* New domain tools can be added without changing the MCP core
* Command Center tools can reuse the same registry and runtime

⸻

16. Final Recommendation

Start with a simple platform-level MCP Server and expose only low-risk ITOM read tools first.

Do not build a separate MCP Server only for ITOM.

The correct long-term design is:

Open MCP Server
  -> ITOM Tools
  -> Command Center Tools
  -> Asset Tools
  -> Automation Tools
  -> Reporting Tools

This gives Trio one reusable AI integration layer for multiple LLM clients and product domains.