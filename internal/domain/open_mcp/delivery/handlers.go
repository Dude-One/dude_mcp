package delivery

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/open-mcp/open-mcp/internal/domain/open_mcp/service"
)

// Handlers holds the MCP tool handler functions.
// Each handler is thin: validate input → call service → format result.
type Handlers struct {
	svc service.MCPService
}

func NewHandlers(svc service.MCPService) *Handlers {
	return &Handlers{svc: svc}
}

// HandleListSources returns all registered domain sources.
func (h *Handlers) HandleListSources(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	sources, err := h.svc.ListSources(ctx)
	if err != nil {
		return nil, fmt.Errorf("list_sources: %w", err)
	}

	type sourceInfo struct {
		Name           string `json:"name"`
		PlatformDomain string `json:"platform_domain"`
		Description    string `json:"description"`
		Type           string `json:"type"`
		BaseURL        string `json:"base_url"`
	}

	out := make([]sourceInfo, 0, len(sources))
	for _, s := range sources {
		out = append(out, sourceInfo{
			Name:           s.Name,
			PlatformDomain: s.PlatformDomain,
			Description:    s.Description,
			Type:           string(s.Type),
			BaseURL:        s.BaseURL,
		})
	}

	data, err := json.MarshalIndent(out, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("encoding sources: %w", err)
	}

	return mcp.NewToolResultText(string(data)), nil
}

// HandleGetAPISpec returns the live Swagger JSON for a named source.
func (h *Handlers) HandleGetAPISpec(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	sourceName, err := req.RequireString("source")
	if err != nil {
		return nil, fmt.Errorf("get_api_spec: %w", err)
	}

	spec, err := h.svc.GetAPISpec(ctx, sourceName)
	if err != nil {
		return nil, fmt.Errorf("get_api_spec %q: %w", sourceName, err)
	}

	return mcp.NewToolResultText(string(spec)), nil
}

// HandleListContextDocs returns all available doc paths for a named source.
func (h *Handlers) HandleListContextDocs(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	sourceName, err := req.RequireString("source")
	if err != nil {
		return nil, fmt.Errorf("list_context_docs: %w", err)
	}

	paths, err := h.svc.ListContextDocs(ctx, sourceName)
	if err != nil {
		return nil, fmt.Errorf("list_context_docs %q: %w", sourceName, err)
	}

	data, err := json.MarshalIndent(paths, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("encoding doc list: %w", err)
	}

	return mcp.NewToolResultText(string(data)), nil
}

// HandleReadContextDoc returns the content of one context doc.
func (h *Handlers) HandleReadContextDoc(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	sourceName, err := req.RequireString("source")
	if err != nil {
		return nil, fmt.Errorf("read_context_doc: %w", err)
	}
	docPath, err := req.RequireString("path")
	if err != nil {
		return nil, fmt.Errorf("read_context_doc: %w", err)
	}

	content, err := h.svc.ReadContextDoc(ctx, sourceName, docPath)
	if err != nil {
		return nil, fmt.Errorf("read_context_doc %q/%s: %w", sourceName, docPath, err)
	}

	return mcp.NewToolResultText(string(content)), nil
}

// HandleCallAPI proxies any HTTP method to a domain API endpoint.
func (h *Handlers) HandleCallAPI(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	sourceName, err := req.RequireString("source")
	if err != nil {
		return nil, fmt.Errorf("call_api: %w", err)
	}
	method, err := req.RequireString("method")
	if err != nil {
		return nil, fmt.Errorf("call_api: %w", err)
	}
	apiPath, err := req.RequireString("path")
	if err != nil {
		return nil, fmt.Errorf("call_api: %w", err)
	}

	tenantID := req.GetString("tenant_id", "")
	requestBody := req.GetString("body", "")
	// session_token is injected by MCP middleware from the user's IDE credentials.
	sessionToken := req.GetString("session_token", "")

	body, status, err := h.svc.CallAPI(ctx, sourceName, method, apiPath, tenantID, requestBody, sessionToken)
	if err != nil {
		return nil, fmt.Errorf("call_api %s %s: %w", method, apiPath, err)
	}

	result := map[string]any{
		"status": status,
		"body":   string(body),
	}
	data, _ := json.MarshalIndent(result, "", "  ")
	return mcp.NewToolResultText(string(data)), nil
}
