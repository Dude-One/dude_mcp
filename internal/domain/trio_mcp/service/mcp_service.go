package service

import (
	"context"

	"github.com/trio/triov2-mcp/internal/domain/trio_mcp/entity"
)

// MCPService is the interface the delivery layer calls.
// Each method maps 1:1 to an MCP tool exposed to the LLM.
// sessionToken is the user's active panel session resolved from IDE credentials per request.
type MCPService interface {
	ListSources(ctx context.Context) ([]entity.Source, error)
	GetAPISpec(ctx context.Context, sourceName string) ([]byte, error)
	ListContextDocs(ctx context.Context, sourceName string) ([]string, error)
	ReadContextDoc(ctx context.Context, sourceName, docPath string) ([]byte, error)
	// CallAPI proxies any HTTP request. sessionToken carries the user's identity.
	CallAPI(ctx context.Context, sourceName, method, path, tenantID, body, sessionToken string) ([]byte, int, error)
}
