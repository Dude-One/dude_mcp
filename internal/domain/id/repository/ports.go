package repository

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/open-mcp/open-mcp/internal/domain/id/entity"
)

// UserRepository fetches users from the mock IdP store.
type UserRepository interface {
	FindByUsername(ctx context.Context, username string) (*entity.User, error)
	FindByID(ctx context.Context, id uuid.UUID) (*entity.User, error)
	FindAll(ctx context.Context) ([]*entity.User, error)
	ExistsByUsername(ctx context.Context, username string) (bool, error)
	Create(ctx context.Context, user *entity.User) error
	UpdateContextNotes(ctx context.Context, userID uuid.UUID, notes *string) error
	UpdateRole(ctx context.Context, userID uuid.UUID, role string) error
	UpdateAllowedProjects(ctx context.Context, userID uuid.UUID, projects []string) error
}

// TenantRepository resolves tenants for new user registration.
type TenantRepository interface {
	FindByName(ctx context.Context, name string) (uuid.UUID, error)
}

// SessionRepository manages opaque panel browser sessions.
type SessionRepository interface {
	CreateSession(ctx context.Context, userID uuid.UUID, token string, expiresAt time.Time) error
	FindSession(ctx context.Context, token string) (*entity.User, error)
	RevokeSession(ctx context.Context, token string) error
}
