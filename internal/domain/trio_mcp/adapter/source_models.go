package adapter

import (
	"time"

	"github.com/google/uuid"
)

// MCPSource is the GORM model for the mcp_sources table.
// Stores URL config only — no credentials.
type MCPSource struct {
	ID             uuid.UUID `gorm:"type:uuid;primaryKey"`
	Name           string    `gorm:"size:255;not null;uniqueIndex"`
	PlatformDomain string    `gorm:"size:64;not null"`
	Description    string    `gorm:"type:text;not null;default:''"`
	Type           string    `gorm:"size:32;not null;default:api"`
	BaseURL        string    `gorm:"type:text;not null"`
	SwaggerPath    string    `gorm:"type:text;not null;default:/swagger/doc.json"`
	ContextPath    string    `gorm:"type:text;not null;default:/mcp/context"`
	IsActive       bool      `gorm:"not null;default:true"`
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

func (MCPSource) TableName() string { return "mcp_sources" }

// MigrateAdminTables runs AutoMigrate for MCP admin tables.
func MigrateAdminTables(db interface{ AutoMigrate(...interface{}) error }) error {
	return db.AutoMigrate(&MCPSource{})
}
