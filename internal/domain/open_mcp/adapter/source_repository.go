package adapter

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/open-mcp/open-mcp/internal/domain/open_mcp/entity"
	"gorm.io/gorm"
)

// PostgresAdminSourceRepository implements AdminSourceRepository via GORM.
type PostgresAdminSourceRepository struct{ db *gorm.DB }

func NewPostgresAdminSourceRepository(db *gorm.DB) *PostgresAdminSourceRepository {
	return &PostgresAdminSourceRepository{db: db}
}

func (r *PostgresAdminSourceRepository) ListSources(ctx context.Context) ([]entity.Source, error) {
	var rows []MCPSource
	if err := r.db.WithContext(ctx).Order("name").Find(&rows).Error; err != nil {
		return nil, fmt.Errorf("listing sources: %w", err)
	}
	out := make([]entity.Source, 0, len(rows))
	for _, row := range rows {
		out = append(out, rowToSource(row))
	}
	return out, nil
}

func (r *PostgresAdminSourceRepository) GetSourceByID(ctx context.Context, id uuid.UUID) (*entity.Source, error) {
	var row MCPSource
	err := r.db.WithContext(ctx).Where("id = ?", id).First(&row).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, fmt.Errorf("source %q not found", id)
		}
		return nil, fmt.Errorf("getting source: %w", err)
	}
	s := rowToSource(row)
	return &s, nil
}

func (r *PostgresAdminSourceRepository) CreateSource(ctx context.Context, src entity.Source) error {
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
	return r.db.WithContext(ctx).Create(&row).Error
}

func (r *PostgresAdminSourceRepository) UpdateSource(ctx context.Context, id uuid.UUID, src entity.Source) error {
	updates := map[string]any{
		"name":            src.Name,
		"platform_domain": src.PlatformDomain,
		"description":     src.Description,
		"type":            string(src.Type),
		"base_url":        src.BaseURL,
		"swagger_path":    src.SwaggerPath,
		"context_path":    src.ContextPath,
	}
	return r.db.WithContext(ctx).Model(&MCPSource{}).Where("id = ?", id).Updates(updates).Error
}

func (r *PostgresAdminSourceRepository) DeleteSource(ctx context.Context, id uuid.UUID) error {
	return r.db.WithContext(ctx).Where("id = ?", id).Delete(&MCPSource{}).Error
}

func (r *PostgresAdminSourceRepository) ToggleSource(ctx context.Context, id uuid.UUID, active bool) error {
	return r.db.WithContext(ctx).Model(&MCPSource{}).
		Where("id = ?", id).
		Update("is_active", active).Error
}

func rowToSource(row MCPSource) entity.Source {
	return entity.Source{
		ID:             row.ID,
		Name:           row.Name,
		PlatformDomain: row.PlatformDomain,
		Description:    row.Description,
		Type:           entity.SourceType(row.Type),
		BaseURL:        row.BaseURL,
		SwaggerPath:    row.SwaggerPath,
		ContextPath:    row.ContextPath,
		IsActive:       row.IsActive,
	}
}
