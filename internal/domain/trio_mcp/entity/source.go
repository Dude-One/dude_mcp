package entity

import "github.com/google/uuid"

// SourceType distinguishes API sources (full swagger + call_api) from
// context-only sources (docs only, no API proxy).
type SourceType string

const (
	SourceTypeAPI     SourceType = "api"
	SourceTypeContext SourceType = "context"
)

// Source is one registered domain service — just URL config.
// Authentication is handled by the MCP middleware using the user's
// credentials from their IDE config (username + password per request).
type Source struct {
	ID             uuid.UUID  `json:"id"`
	Name           string     `json:"name"`
	PlatformDomain string     `json:"platform_domain"` // id, infra, device, tenant, ...
	Description    string     `json:"description"`
	Type           SourceType `json:"type"`
	BaseURL        string     `json:"base_url"`
	SwaggerPath    string     `json:"swagger_path"` // e.g. /swagger/doc.json
	ContextPath    string     `json:"context_path"` // e.g. /mcp/context
	IsActive       bool       `json:"is_active"`
}

// CachedSpec holds a fetched Swagger spec with its TTL metadata.
type CachedSpec struct {
	SourceName string
	JSON       []byte
	FetchedAt  int64 // unix seconds
}
