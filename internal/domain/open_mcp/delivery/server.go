package delivery

import (
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

// NewMCPServer creates and registers all MCP tools.
// Each tool maps to one handler method — the LLM calls these by name.
func NewMCPServer(h *Handlers) *server.MCPServer {
	s := server.NewMCPServer(
		"Open MCP",
		"1.0.0",
		server.WithToolCapabilities(true),
	)

	// list_sources — what domains are registered?
	s.AddTool(
		mcp.NewTool("list_sources",
			mcp.WithDescription("List all registered Open MCP domain sources (panel, infra, device, etc.). Call this first to discover what APIs and context docs are available."),
		),
		h.HandleListSources,
	)

	// get_api_spec — fetch the swagger for a domain
	s.AddTool(
		mcp.NewTool("get_api_spec",
			mcp.WithDescription("Fetch the live OpenAPI/Swagger spec for a domain source. Use this to understand what endpoints are available before calling call_api."),
			mcp.WithString("source",
				mcp.Required(),
				mcp.Description("Source name from list_sources, e.g. open-mcp-panel"),
			),
		),
		h.HandleGetAPISpec,
	)

	// list_context_docs — what docs does a domain have?
	s.AddTool(
		mcp.NewTool("list_context_docs",
			mcp.WithDescription("List all available context doc paths for a domain source. Use this to discover what reference material is available."),
			mcp.WithString("source",
				mcp.Required(),
				mcp.Description("Source name from list_sources, e.g. open-mcp-panel"),
			),
		),
		h.HandleListContextDocs,
	)

	// read_context_doc — read one doc
	s.AddTool(
		mcp.NewTool("read_context_doc",
			mcp.WithDescription("Read the content of a context doc from a domain source. Use this to understand domain concepts before making API calls."),
			mcp.WithString("source",
				mcp.Required(),
				mcp.Description("Source name from list_sources, e.g. open-mcp-panel"),
			),
			mcp.WithString("path",
				mcp.Required(),
				mcp.Description("Doc path from list_context_docs, e.g. README.md or guides/devices.md"),
			),
		),
		h.HandleReadContextDoc,
	)

	// call_api — proxy any method to a domain API
	s.AddTool(
		mcp.NewTool("call_api",
			mcp.WithDescription("Call a domain API endpoint. Supports GET, HEAD, POST, PUT, PATCH, DELETE. Always call get_api_spec first to know the available endpoints and required request bodies."),
			mcp.WithString("source",
				mcp.Required(),
				mcp.Description("Source name from list_sources, e.g. open-mcp-agentless"),
			),
			mcp.WithString("method",
				mcp.Required(),
				mcp.Description("HTTP method: GET, HEAD, POST, PUT, PATCH or DELETE"),
			),
			mcp.WithString("path",
				mcp.Required(),
				mcp.Description("API path to call, e.g. /panel/me"),
			),
			mcp.WithString("tenant_id",
				mcp.Description("Tenant ID to scope the request. Injected as X-Tenant-ID header."),
			),
			mcp.WithString("body",
				mcp.Description("JSON request body for POST, PUT, PATCH requests. Leave empty for GET, HEAD, DELETE."),
			),
		),
		h.HandleCallAPI,
	)

	return s
}
