package adapter

import (
	"fmt"
	"os"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func OpenFromEnv() (*gorm.DB, error) {
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		dsn = fmt.Sprintf(
			"host=%s user=%s password=%s dbname=%s port=%s sslmode=disable",
			envOr("AUTH_DB_HOST", "localhost"),
			envOr("AUTH_DB_USER", "trio_mcp"),
			envOr("AUTH_DB_PASSWORD", "trio_mcp_dev"),
			envOr("AUTH_DB_NAME", "trio_mcp_provider"),
			envOr("AUTH_DB_PORT", "5433"),
		)
	}
	logLevel := logger.Warn
	if os.Getenv("GORM_LOG") == "debug" {
		logLevel = logger.Info
	}
	return gorm.Open(postgres.Open(dsn), &gorm.Config{
		Logger:                                   logger.Default.LogMode(logLevel),
		DisableForeignKeyConstraintWhenMigrating: true,
	})
}

func Migrate(db *gorm.DB) error {
	return db.AutoMigrate(
		&Tenant{},
		&User{},
		&UserSession{},
	)
}

func envOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
