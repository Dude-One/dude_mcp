package service

import (
	"context"
	"fmt"
	"strings"

	"github.com/Dude-One/dude_mcp/internal/domain/dude_mcp/entity"
	"github.com/Dude-One/dude_mcp/internal/domain/dude_mcp/repository"
)

var validMethods = map[string]bool{
	"GET": true, "HEAD": true, "POST": true,
	"PUT": true, "PATCH": true, "DELETE": true,
}

type MCPServiceImpl struct {
	registry repository.SourceRegistry
	swagger  repository.SwaggerReader
	context  repository.ContextReader
	caller   repository.APICaller
}

func NewMCPService(
	registry repository.SourceRegistry,
	swagger repository.SwaggerReader,
	context repository.ContextReader,
	caller repository.APICaller,
) *MCPServiceImpl {
	return &MCPServiceImpl{registry: registry, swagger: swagger, context: context, caller: caller}
}

func (s *MCPServiceImpl) ListSources(ctx context.Context) ([]entity.Source, error) {
	return s.registry.ListSources(ctx)
}

func (s *MCPServiceImpl) GetAPISpec(ctx context.Context, sourceName string) ([]byte, error) {
	src, err := s.registry.GetSource(ctx, sourceName)
	if err != nil {
		return nil, fmt.Errorf("source %q not found: %w", sourceName, err)
	}
	return s.swagger.GetSpec(ctx, *src)
}

func (s *MCPServiceImpl) ListContextDocs(ctx context.Context, sourceName string) ([]string, error) {
	src, err := s.registry.GetSource(ctx, sourceName)
	if err != nil {
		return nil, fmt.Errorf("source %q not found: %w", sourceName, err)
	}
	return s.context.ListDocs(ctx, *src)
}

func (s *MCPServiceImpl) ReadContextDoc(ctx context.Context, sourceName, docPath string) ([]byte, error) {
	if strings.Contains(docPath, "..") {
		return nil, fmt.Errorf("invalid doc path: path traversal not allowed")
	}
	src, err := s.registry.GetSource(ctx, sourceName)
	if err != nil {
		return nil, fmt.Errorf("source %q not found: %w", sourceName, err)
	}
	return s.context.ReadDoc(ctx, *src, docPath)
}

func (s *MCPServiceImpl) CallAPI(ctx context.Context, sourceName, method, path, tenantID, body, sessionToken string) ([]byte, int, error) {
	method = strings.ToUpper(method)
	if !validMethods[method] {
		return nil, 405, fmt.Errorf("unrecognised method %q", method)
	}
	if strings.Contains(path, "..") {
		return nil, 400, fmt.Errorf("invalid path: path traversal not allowed")
	}
	src, err := s.registry.GetSource(ctx, sourceName)
	if err != nil {
		return nil, 404, fmt.Errorf("source %q not found: %w", sourceName, err)
	}
	return s.caller.Call(ctx, *src, method, path, tenantID, body, sessionToken)
}
