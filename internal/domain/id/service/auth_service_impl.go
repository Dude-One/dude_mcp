package service

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/trio/triov2-mcp/internal/domain/id/entity"
	"github.com/trio/triov2-mcp/internal/domain/id/repository"
)

// AuthServiceImpl wires panel session and user repositories.
type AuthServiceImpl struct {
	users    repository.UserRepository
	tenants  repository.TenantRepository
	sessions repository.SessionRepository
}

func NewAuthService(
	users repository.UserRepository,
	tenants repository.TenantRepository,
	sessions repository.SessionRepository,
) *AuthServiceImpl {
	return &AuthServiceImpl{
		users:    users,
		tenants:  tenants,
		sessions: sessions,
	}
}

func (s *AuthServiceImpl) Login(ctx context.Context, username, credential string) (string, error) {
	user, err := s.users.FindByUsername(ctx, username)
	if err != nil {
		return "", errors.New("invalid credentials")
	}
	if !user.ValidateCredential(credential) {
		return "", errors.New("invalid credentials")
	}
	sessionToken := entity.GenerateOpaqueToken("sess_")
	if err := s.sessions.CreateSession(ctx, user.ID, sessionToken, time.Now().Add(24*time.Hour)); err != nil {
		return "", err
	}
	return sessionToken, nil
}

func (s *AuthServiceImpl) SignUp(ctx context.Context, username, credential, displayName string) (string, error) {
	username = strings.TrimSpace(username)
	credential = strings.TrimSpace(credential)
	displayName = strings.TrimSpace(displayName)

	exists, err := s.users.ExistsByUsername(ctx, username)
	if err != nil {
		return "", err
	}
	if exists {
		return "", errors.New("username already taken")
	}

	tenantID, err := s.tenants.FindByName(ctx, "default")
	if err != nil {
		return "", err
	}

	user, err := entity.NewPasswordUser(tenantID, username, credential, displayName)
	if err != nil {
		return "", err
	}

	if err := s.users.Create(ctx, user); err != nil {
		return "", err
	}

	return s.Login(ctx, username, credential)
}

func (s *AuthServiceImpl) Logout(ctx context.Context, sessionToken string) error {
	return s.sessions.RevokeSession(ctx, sessionToken)
}

func (s *AuthServiceImpl) ValidateSession(ctx context.Context, sessionToken string) (*entity.SessionClaims, error) {
	user, err := s.sessions.FindSession(ctx, sessionToken)
	if err != nil {
		return nil, err
	}
	return &entity.SessionClaims{
		UserID:          user.ID,
		TenantID:        user.TenantID,
		Username:        user.Username,
		DisplayName:     user.DisplayName,
		AllowedProjects: user.AllowedProjects,
		Role:            user.Role,
	}, nil
}

func (s *AuthServiceImpl) GetContextNotes(ctx context.Context, sessionToken string) (string, error) {
	claims, err := s.ValidateSession(ctx, sessionToken)
	if err != nil {
		return "", err
	}
	user, err := s.users.FindByID(ctx, claims.UserID)
	if err != nil {
		return "", err
	}
	if user.ContextNotes == nil {
		return "", nil
	}
	return *user.ContextNotes, nil
}

func (s *AuthServiceImpl) SaveContextNotes(ctx context.Context, sessionToken, notes string) error {
	claims, err := s.ValidateSession(ctx, sessionToken)
	if err != nil {
		return err
	}
	if notes == "" {
		return s.users.UpdateContextNotes(ctx, claims.UserID, nil)
	}
	return s.users.UpdateContextNotes(ctx, claims.UserID, &notes)
}
