package domain

import (
	"context"
	"time"

	"github.com/google/uuid"
)

// =============================================================
// UserRepository — kontrak ke Postgres
// =============================================================

type UserRepository interface {
	// Auth
	Create(ctx context.Context, user *User) error
	GetByID(ctx context.Context, id uuid.UUID) (*User, error)
	GetByEmail(ctx context.Context, email string) (*User, error)
	GetByGoogleID(ctx context.Context, googleID string) (*User, error)
	Update(ctx context.Context, user *User) error

	// Profile
	CreateProfile(ctx context.Context, profile *UserProfile) error
	GetProfileByUserID(ctx context.Context, userID uuid.UUID) (*UserProfile, error)
	UpdateProfile(ctx context.Context, profile *UserProfile) error

	// Address
	CreateAddress(ctx context.Context, address *UserAddress) error
	GetAddressByID(ctx context.Context, id uuid.UUID) (*UserAddress, error)
	GetAddressesByUserID(ctx context.Context, userID uuid.UUID) ([]*UserAddress, error)
	UpdateAddress(ctx context.Context, address *UserAddress) error
	DeleteAddress(ctx context.Context, id uuid.UUID, userID uuid.UUID) error
	UnsetDefaultAddress(ctx context.Context, userID uuid.UUID) error
}

// =============================================================
// TokenRepository — kontrak ke Redis
// =============================================================

type TokenRepository interface {
	// Refresh token (Redis sebagai primary store)
	SaveRefreshToken(ctx context.Context, token *RefreshToken) error
	GetRefreshToken(ctx context.Context, tokenHash string) (*RefreshToken, error)
	RevokeRefreshToken(ctx context.Context, tokenHash string) error
	RevokeAllUserTokens(ctx context.Context, userID uuid.UUID) error

	// Access token blacklist
	BlacklistAccessToken(ctx context.Context, tokenHash string, claims *Claims) error
	IsAccessTokenBlacklisted(ctx context.Context, tokenHash string) (bool, error)

	// Rate limiting (digunakan middleware via RedisRateLimiter, bukan langsung tokenRepo)
	IncrRateLimit(ctx context.Context, key string, window time.Duration) (int64, time.Time, error)
	GetRateLimit(ctx context.Context, key string) (int64, error)
	ResetRateLimit(ctx context.Context, key string) error
}
