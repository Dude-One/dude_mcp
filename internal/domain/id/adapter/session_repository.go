package adapter

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/Dude-One/dude_mcp/internal/domain/id/entity"
	"gorm.io/gorm"
)

type PostgresSessionRepository struct{ db *gorm.DB }

func NewPostgresSessionRepository(db *gorm.DB) *PostgresSessionRepository {
	return &PostgresSessionRepository{db: db}
}

func (r *PostgresSessionRepository) CreateSession(
	ctx context.Context,
	userID uuid.UUID,
	token string,
	expiresAt time.Time,
) error {
	s := UserSession{
		ID:           uuid.New(),
		UserID:       userID,
		SessionToken: token,
		ExpiresAt:    expiresAt,
	}
	return r.db.WithContext(ctx).Create(&s).Error
}

// FindSession returns the owning User if the session token is valid and not expired.
func (r *PostgresSessionRepository) FindSession(ctx context.Context, token string) (*entity.User, error) {
	var s UserSession
	err := r.db.WithContext(ctx).
		Where("session_token = ? AND revoked_at IS NULL AND expires_at > NOW()", token).
		First(&s).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("session not found or expired")
		}
		return nil, err
	}

	var u User
	if err := r.db.WithContext(ctx).Where("id = ? AND is_active = true", s.UserID).First(&u).Error; err != nil {
		return nil, err
	}
	return userToDomain(&u), nil
}

func (r *PostgresSessionRepository) RevokeSession(ctx context.Context, token string) error {
	return r.db.WithContext(ctx).
		Model(&UserSession{}).
		Where("session_token = ?", token).
		Update("revoked_at", time.Now()).Error
}
