package repository

import (
	"context"

	"github.com/google/uuid"
	"github.com/Dude-One/dude_mcp/internal/domain/dude_mcp/entity"
)

// SourceRegistry lists and looks up registered domain sources.
type SourceRegistry interface {
	ListSources(ctx context.Context) ([]entity.Source, error)
	GetSource(ctx context.Context, name string) (*entity.Source, error)
}

// SwaggerReader fetches and caches the OpenAPI spec for a source.
type SwaggerReader interface {
	GetSpec(ctx context.Context, source entity.Source) ([]byte, error)
}

// ContextReader fetches context docs from a domain's /mcp/context endpoint.
type ContextReader interface {
	ListDocs(ctx context.Context, source entity.Source) ([]string, error)
	ReadDoc(ctx context.Context, source entity.Source, path string) ([]byte, error)
}

// APICaller proxies HTTP calls to a domain API.
// sessionToken is the user's active panel session — resolved per request from IDE credentials.
// body is the raw JSON body for POST/PUT/PATCH (empty for GET/HEAD/DELETE).
type APICaller interface {
	Call(ctx context.Context, source entity.Source, method, path, tenantID, body, sessionToken string) ([]byte, int, error)
}

// AdminSourceRepository manages sources stored in the DB (URL config only — no credentials).
type AdminSourceRepository interface {
	ListSources(ctx context.Context) ([]entity.Source, error)
	GetSourceByID(ctx context.Context, id uuid.UUID) (*entity.Source, error)
	CreateSource(ctx context.Context, src entity.Source) error
	UpdateSource(ctx context.Context, id uuid.UUID, src entity.Source) error
	DeleteSource(ctx context.Context, id uuid.UUID) error
	ToggleSource(ctx context.Context, id uuid.UUID, active bool) error
}
