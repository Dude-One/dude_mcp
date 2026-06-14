package adapter

import (
	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

var defaultTenantID = uuid.MustParse("00000000-0000-4000-8000-000000000001")

// Seed ensures the default tenant exists. Safe to call multiple times.
// Users are no longer seeded — the first admin is created through the setup flow.
func Seed(db *gorm.DB) error {
	tenant := Tenant{ID: defaultTenantID, Name: "default"}
	return db.Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "name"}},
		DoNothing: true,
	}).Create(&tenant).Error
}
