package adapter

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// SeedFromYAML imports sources from a YAMLRegistry into the DB if mcp_sources
// is empty. Runs once on first startup — DB is the source of truth after that.
func SeedFromYAML(ctx context.Context, db *gorm.DB, yaml *YAMLRegistry) error {
	dbReg := NewDBRegistry(db)

	empty, err := dbReg.IsEmpty(ctx)
	if err != nil {
		return fmt.Errorf("checking sources table: %w", err)
	}
	if !empty {
		return nil // already seeded
	}

	sources, err := yaml.ListSources(ctx)
	if err != nil {
		return fmt.Errorf("reading yaml registry: %w", err)
	}

	for _, src := range sources {
		row := MCPSource{
			ID:             uuid.New(),
			Name:           src.Name,
			PlatformDomain: src.PlatformDomain,
			Description:    src.Description,
			Type:           string(src.Type),
			BaseURL:        src.BaseURL,
			SwaggerPath:    src.SwaggerPath,
			ContextPath:    src.ContextPath,
			IsActive:       true,
		}
		if err := db.WithContext(ctx).Create(&row).Error; err != nil {
			return fmt.Errorf("seeding source %q: %w", src.Name, err)
		}
	}

	return nil
}
