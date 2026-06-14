package adapter

import (
	"time"

	"github.com/google/uuid"
	"github.com/lib/pq"
)

const (
	CredentialTypePassword = "password"
	CredentialTypeToken    = "token"
)

type Tenant struct {
	ID        uuid.UUID `gorm:"type:uuid;primaryKey"`
	Name      string    `gorm:"size:255;not null;uniqueIndex"`
	CreatedAt time.Time
}

func (Tenant) TableName() string { return "tenants" }

type User struct {
	ID              uuid.UUID      `gorm:"type:uuid;primaryKey"`
	TenantID        uuid.UUID      `gorm:"type:uuid;not null;index"`
	Username        string         `gorm:"size:255;not null;uniqueIndex"`
	Credential      string         `gorm:"type:text;not null"`
	CredentialType  string         `gorm:"size:32;not null;default:password"`
	DisplayName     string         `gorm:"size:255"`
	AllowedProjects pq.StringArray `gorm:"type:text[];not null;default:'{}'"`
	ContextNotes    *string        `gorm:"type:text"`
	Role            string         `gorm:"size:32;not null;default:user"`
	IsActive        bool           `gorm:"not null;default:true"`
	CreatedAt       time.Time
	UpdatedAt       time.Time
}

func (User) TableName() string { return "users" }

type UserSession struct {
	ID           uuid.UUID `gorm:"type:uuid;primaryKey"`
	UserID       uuid.UUID `gorm:"type:uuid;not null;index"`
	SessionToken string    `gorm:"type:text;not null;uniqueIndex"`
	ExpiresAt    time.Time `gorm:"not null"`
	CreatedAt    time.Time
	RevokedAt    *time.Time
}

func (UserSession) TableName() string { return "user_sessions" }
