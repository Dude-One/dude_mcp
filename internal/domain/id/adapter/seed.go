package adapter

import (
	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

var (
	defaultTenantID = uuid.MustParse("00000000-0000-4000-8000-000000000001")
	adminUserID     = uuid.MustParse("00000000-0000-4000-8000-000000000010")
	serviceUserID   = uuid.MustParse("00000000-0000-4000-8000-000000000011")
)

// Seed inserts local dev data. Safe to call multiple times (upsert on conflict).
func Seed(db *gorm.DB) error {
	tenant := Tenant{ID: defaultTenantID, Name: "default"}
	if err := db.Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "name"}},
		DoNothing: true,
	}).Create(&tenant).Error; err != nil {
		return err
	}

	users := []User{
		{
			ID:              adminUserID,
			TenantID:        defaultTenantID,
			Username:        "javad.kz",
			Credential:      "ecdb01d2b4488de286cc985a0d06ce07bb54f66f24cf26a49b4e2e171b7f137f",
			CredentialType:  CredentialTypePassword,
			DisplayName:     "Javad",
			AllowedProjects: []string{"trio-mcp", "trio-servermanagement"},
			Role:            "admin",
			IsActive:        true,
		},
		{
			ID:              serviceUserID,
			TenantID:        defaultTenantID,
			Username:        "service-bot",
			Credential:      "dev-service-token-change-me",
			CredentialType:  CredentialTypeToken,
			DisplayName:     "Service Bot",
			AllowedProjects: []string{"trio-mcp"},
			Role:            "user",
			IsActive:        true,
		},
	}
	for i := range users {
		u := users[i]
		if err := db.Clauses(clause.OnConflict{
			Columns:   []clause.Column{{Name: "username"}},
			DoNothing: true,
		}).Create(&u).Error; err != nil {
			return err
		}
	}

	return nil
}
