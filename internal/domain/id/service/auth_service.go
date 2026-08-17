package service

import (
	"context"

	"github.com/Dude-One/dude_mcp/internal/domain/id/entity"
)

// AuthService is the application port for panel login and context notes.
// Consumed by the delivery layer; implemented by AuthServiceImpl.
type AuthService interface {
	NeedsSetup(ctx context.Context) (bool, error)
	SetupFirstAdmin(ctx context.Context, username, credential, displayName string) (sessionToken string, err error)
	Login(ctx context.Context, username, credential string) (sessionToken string, err error)
	SignUp(ctx context.Context, username, credential, displayName string) (sessionToken string, err error)
	Logout(ctx context.Context, sessionToken string) error
	ValidateSession(ctx context.Context, sessionToken string) (*entity.SessionClaims, error)
	GetContextNotes(ctx context.Context, sessionToken string) (notes string, err error)
	SaveContextNotes(ctx context.Context, sessionToken, notes string) error
}
