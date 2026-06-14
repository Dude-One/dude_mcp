package adapter

import (
	"context"
	"fmt"
	"sync"

	"github.com/trio/triov2-mcp/internal/domain/trio_mcp/entity"
	"gorm.io/gorm"
)

// DBRegistry reads domain sources from the mcp_sources table.
// Stores URL config only — no credentials.
// Results are cached; call Reload() to invalidate after an admin panel change.
type DBRegistry struct {
	db    *gorm.DB
	mu    sync.RWMutex
	cache map[string]entity.Source
}

func NewDBRegistry(db *gorm.DB) *DBRegistry {
	return &DBRegistry{db: db}
}

// Reload invalidates the cache. Next call re-fetches from DB.
func (r *DBRegistry) Reload() error {
	r.mu.Lock()
	r.cache = nil
	r.mu.Unlock()
	return nil
}

// IsEmpty reports whether mcp_sources has any rows (used for first-run seeding).
func (r *DBRegistry) IsEmpty(ctx context.Context) (bool, error) {
	var count int64
	if err := r.db.WithContext(ctx).Model(&MCPSource{}).Count(&count).Error; err != nil {
		return false, fmt.Errorf("counting sources: %w", err)
	}
	return count == 0, nil
}

func (r *DBRegistry) ListSources(ctx context.Context) ([]entity.Source, error) {
	if err := r.loadIfNeeded(ctx); err != nil {
		return nil, err
	}
	r.mu.RLock()
	defer r.mu.RUnlock()
	out := make([]entity.Source, 0, len(r.cache))
	for _, s := range r.cache {
		out = append(out, s)
	}
	return out, nil
}

func (r *DBRegistry) GetSource(ctx context.Context, name string) (*entity.Source, error) {
	if err := r.loadIfNeeded(ctx); err != nil {
		return nil, err
	}
	r.mu.RLock()
	defer r.mu.RUnlock()
	s, ok := r.cache[name]
	if !ok {
		return nil, fmt.Errorf("source %q not registered", name)
	}
	return &s, nil
}

func (r *DBRegistry) loadIfNeeded(ctx context.Context) error {
	r.mu.RLock()
	loaded := r.cache != nil
	r.mu.RUnlock()
	if loaded {
		return nil
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.cache != nil {
		return nil
	}
	return r.load(ctx)
}

func (r *DBRegistry) load(ctx context.Context) error {
	var rows []MCPSource
	if err := r.db.WithContext(ctx).
		Where("is_active = true").
		Order("name").
		Find(&rows).Error; err != nil {
		return fmt.Errorf("loading sources from DB: %w", err)
	}

	sources := make(map[string]entity.Source, len(rows))
	for _, row := range rows {
		sources[row.Name] = entity.Source{
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

	r.cache = sources
	return nil
}
