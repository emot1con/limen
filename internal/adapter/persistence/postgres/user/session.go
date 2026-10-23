package user

import (
	"context"
	"errors"
	"net"
	"time"

	domain "limen/internal/domain"
	userdomain "limen/internal/domain/user"

	"gorm.io/gorm"
)

type sessionModel struct {
	ID         string     `gorm:"column:id;primaryKey;default:gen_random_uuid()"`
	UserID     string     `gorm:"column:user_id"`
	TokenHash  string     `gorm:"column:token_hash"`
	CreatedAt  time.Time  `gorm:"column:created_at"`
	LastSeenAt *time.Time `gorm:"column:last_seen_at"`
	ExpiresAt  time.Time  `gorm:"column:expires_at"`
	RevokedAt  *time.Time `gorm:"column:revoked_at"`
	IP         net.IP     `gorm:"column:ip;type:inet"`
	UserAgent  string     `gorm:"column:user_agent"`
}

func (sessionModel) TableName() string {
	return "sessions"
}

type SessionRepository struct {
	db *gorm.DB
}

func NewSessionRepository(db *gorm.DB) *SessionRepository {
	return &SessionRepository{db: db}
}

func (r *SessionRepository) Create(ctx context.Context, session userdomain.Session) (string, error) {
	model := sessionModel{
		ID:         session.ID,
		UserID:     session.UserID,
		TokenHash:  session.TokenHash,
		CreatedAt:  session.CreatedAt,
		LastSeenAt: session.LastSeenAt,
		ExpiresAt:  session.ExpiresAt,
		RevokedAt:  session.RevokedAt,
		IP:         session.IP,
		UserAgent:  session.UserAgent,
	}
	if err := r.db.WithContext(ctx).Create(&model).Error; err != nil {
		return "", err
	}
	return model.ID, nil
}

func (r *SessionRepository) FindByTokenHash(ctx context.Context, tokenHash string) (userdomain.Session, error) {
	var model sessionModel
	if err := r.db.WithContext(ctx).Where("token_hash = ?", tokenHash).First(&model).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return userdomain.Session{}, domain.ErrNotFound
		}
		return userdomain.Session{}, err
	}
	return r.toDomain(model), nil
}

func (r *SessionRepository) Revoke(ctx context.Context, id string, revokedAt time.Time) error {
	result := r.db.WithContext(ctx).Model(&sessionModel{}).Where("id = ?", id).Update("revoked_at", revokedAt)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return domain.ErrNotFound
	}
	return nil
}

func (r *SessionRepository) RevokeAllByUserID(ctx context.Context, userID string, revokedAt time.Time) error {
	return r.db.WithContext(ctx).Model(&sessionModel{}).Where("user_id = ?", userID).Update("revoked_at", revokedAt).Error
}

func (r *SessionRepository) toDomain(model sessionModel) userdomain.Session {
	return userdomain.Session{
		ID:         model.ID,
		UserID:     model.UserID,
		TokenHash:  model.TokenHash,
		CreatedAt:  model.CreatedAt,
		LastSeenAt: model.LastSeenAt,
		ExpiresAt:  model.ExpiresAt,
		RevokedAt:  model.RevokedAt,
		IP:         model.IP,
		UserAgent:  model.UserAgent,
	}
}

var _ userdomain.SessionRepository = (*SessionRepository)(nil)
