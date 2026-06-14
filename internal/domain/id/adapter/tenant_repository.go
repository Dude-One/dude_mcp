package adapter

import (
	"context"
	"errors"
	"strings"

	"github.com/google/uuid"
	"github.com/lib/pq"
	"github.com/open-mcp/open-mcp/internal/domain/id/entity"
	"gorm.io/gorm"
)

type PostgresTenantRepository struct{ db *gorm.DB }

func NewPostgresTenantRepository(db *gorm.DB) *PostgresTenantRepository {
	return &PostgresTenantRepository{db: db}
}

func (r *PostgresTenantRepository) FindByName(ctx context.Context, name string) (uuid.UUID, error) {
	var t Tenant
	err := r.db.WithContext(ctx).Where("name = ?", name).First(&t).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return uuid.Nil, errors.New("tenant not found")
		}
		return uuid.Nil, err
	}
	return t.ID, nil
}

func (r *PostgresUserRepository) ExistsByUsername(ctx context.Context, username string) (bool, error) {
	var count int64
	err := r.db.WithContext(ctx).Model(&User{}).Where("username = ?", username).Count(&count).Error
	return count > 0, err
}

func (r *PostgresUserRepository) Create(ctx context.Context, user *entity.User) error {
	m := &User{
		ID:              user.ID,
		TenantID:        user.TenantID,
		Username:        user.Username,
		Credential:      user.Credential,
		CredentialType:  string(user.CredentialType),
		DisplayName:     user.DisplayName,
		AllowedProjects: pq.StringArray(user.AllowedProjects),
		Role:            string(user.Role),
		IsActive:        user.IsActive,
		CreatedAt:       user.CreatedAt,
		UpdatedAt:       user.UpdatedAt,
	}
	if err := r.db.WithContext(ctx).Create(m).Error; err != nil {
		if strings.Contains(err.Error(), "duplicate key") || strings.Contains(err.Error(), "UNIQUE") {
			return errors.New("username already taken")
		}
		return err
	}
	return nil
}
