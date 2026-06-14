package entity

import (
	"crypto/rand"
	"encoding/hex"

	"github.com/google/uuid"
)

// CredentialType distinguishes how a user authenticates in the mock IdP.
type CredentialType string

const (
	CredentialTypePassword CredentialType = "password"
	CredentialTypeToken    CredentialType = "token"
)

// Role is the user's access level in the panel.
type Role string

const (
	RoleUser  Role = "user"
	RoleAdmin Role = "admin"
)

// SessionClaims is resolved from an active panel session cookie.
type SessionClaims struct {
	UserID          uuid.UUID
	TenantID        uuid.UUID
	Username        string
	DisplayName     string
	AllowedProjects []string
	Role            Role
}

// GenerateOpaqueToken returns a random prefixed hex string (sess_, etc.). Never a JWT.
func GenerateOpaqueToken(prefix string) string {
	b := make([]byte, 32)
	_, _ = rand.Read(b)
	return prefix + hex.EncodeToString(b)
}
