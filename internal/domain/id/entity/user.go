package entity

import (
	"errors"
	"regexp"
	"strings"
	"time"

	"github.com/google/uuid"
)

var usernamePattern = regexp.MustCompile(`^[a-zA-Z0-9_.]{3,32}$`)

// User is the core identity stored in the mock IdP (dude_mcp_db).
type User struct {
	ID              uuid.UUID
	TenantID        uuid.UUID
	Username        string
	Credential      string
	CredentialType  CredentialType
	DisplayName     string
	AllowedProjects []string
	ContextNotes    *string
	Role            Role
	IsActive        bool
	CreatedAt       time.Time
	UpdatedAt       time.Time
}

func NewPasswordUser(tenantID uuid.UUID, username, credential, displayName string) (*User, error) {
	username = strings.TrimSpace(username)
	credential = strings.TrimSpace(credential)
	displayName = strings.TrimSpace(displayName)

	if !usernamePattern.MatchString(username) {
		return nil, errors.New("username must be 3-32 characters: letters, numbers, underscore, dot")
	}
	if len(credential) < 4 {
		return nil, errors.New("credential must be at least 4 characters")
	}
	if displayName == "" {
		displayName = username
	}

	now := time.Now()
	return &User{
		ID:              uuid.New(),
		TenantID:        tenantID,
		Username:        username,
		Credential:      credential,
		CredentialType:  CredentialTypePassword,
		DisplayName:     displayName,
		AllowedProjects: []string{"dude-mcp"},
		Role:            RoleUser,
		IsActive:        true,
		CreatedAt:       now,
		UpdatedAt:       now,
	}, nil
}

func (u *User) ValidateCredential(input string) bool {
	return u.IsActive && u.Credential == input
}
