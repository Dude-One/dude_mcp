package adapter

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/lib/pq"
	"github.com/trio/triov2-mcp/internal/domain/id/entity"
	"gorm.io/gorm"
)

type PostgresUserRepository struct{ db *gorm.DB }

func NewPostgresUserRepository(db *gorm.DB) *PostgresUserRepository {
	return &PostgresUserRepository{db: db}
}

func (r *PostgresUserRepository) FindByUsername(ctx context.Context, username string) (*entity.User, error) {
	var m User
	err := r.db.WithContext(ctx).
		Where("username = ? AND is_active = true", username).
		First(&m).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("user not found")
		}
		return nil, err
	}
	return userToDomain(&m), nil
}

func (r *PostgresUserRepository) FindByID(ctx context.Context, id uuid.UUID) (*entity.User, error) {
	var m User
	err := r.db.WithContext(ctx).
		Where("id = ? AND is_active = true", id).
		First(&m).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("user not found")
		}
		return nil, err
	}
	return userToDomain(&m), nil
}

func (r *PostgresUserRepository) FindAll(ctx context.Context) ([]*entity.User, error) {
	var rows []User
	if err := r.db.WithContext(ctx).Order("username").Find(&rows).Error; err != nil {
		return nil, err
	}
	out := make([]*entity.User, 0, len(rows))
	for i := range rows {
		out = append(out, userToDomain(&rows[i]))
	}
	return out, nil
}

func (r *PostgresUserRepository) UpdateContextNotes(ctx context.Context, userID uuid.UUID, notes *string) error {
	return r.db.WithContext(ctx).Model(&User{}).
		Where("id = ?", userID).
		Update("context_notes", notes).Error
}

func (r *PostgresUserRepository) UpdateRole(ctx context.Context, userID uuid.UUID, role string) error {
	return r.db.WithContext(ctx).Model(&User{}).
		Where("id = ?", userID).
		Update("role", role).Error
}

func (r *PostgresUserRepository) UpdateAllowedProjects(ctx context.Context, userID uuid.UUID, projects []string) error {
	return r.db.WithContext(ctx).Model(&User{}).
		Where("id = ?", userID).
		Update("allowed_projects", pq.StringArray(projects)).Error
}

func userToDomain(m *User) *entity.User {
	return &entity.User{
		ID:              m.ID,
		TenantID:        m.TenantID,
		Username:        m.Username,
		Credential:      m.Credential,
		CredentialType:  entity.CredentialType(m.CredentialType),
		DisplayName:     m.DisplayName,
		AllowedProjects: []string(m.AllowedProjects),
		ContextNotes:    m.ContextNotes,
		Role:            entity.Role(m.Role),
		IsActive:        m.IsActive,
		CreatedAt:       m.CreatedAt,
		UpdatedAt:       m.UpdatedAt,
	}
}
