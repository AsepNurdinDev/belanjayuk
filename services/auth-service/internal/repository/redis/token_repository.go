package redis

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"

	"github.com/AsepNurdinDev/belanjayuk/services/auth-service/internal/domain"
)

// =============================================================
// TokenRepository — implementasi domain.TokenRepository
// Refresh token disimpan di Redis sebagai primary store
// Access token blacklist disimpan di Redis dengan TTL dinamis
// =============================================================

const (
	blacklistPrefix  = "blacklist:access:"  // blacklist:access:<token_hash>
	refreshPrefix    = "refresh:"           // refresh:<token_hash>
	userTokensPrefix = "user:tokens:"       // user:tokens:<user_id>
)

type TokenRepository struct {
	rdb *redis.Client
}

func NewTokenRepository(rdb *redis.Client) *TokenRepository {
	return &TokenRepository{rdb: rdb}
}

// =============================================================
// REFRESH TOKEN
// =============================================================

type refreshTokenData struct {
	UserID    string    `json:"user_id"`
	TokenHash string    `json:"token_hash"`
	ExpiresAt time.Time `json:"expires_at"`
}

func (r *TokenRepository) SaveRefreshToken(ctx context.Context, token *domain.RefreshToken) error {
	data := refreshTokenData{
		UserID:    token.UserID.String(),
		TokenHash: token.TokenHash,
		ExpiresAt: token.ExpiresAt,
	}

	payload, err := json.Marshal(data)
	if err != nil {
		return fmt.Errorf("marshal refresh token: %w", err)
	}

	ttl := time.Until(token.ExpiresAt)
	if ttl <= 0 {
		return domain.ErrExpiredToken
	}

	key := refreshPrefix + token.TokenHash

	pipe := r.rdb.Pipeline()
	pipe.Set(ctx, key, payload, ttl)
	// Track semua token milik user (untuk logout semua device)
	pipe.SAdd(ctx, userTokensPrefix+token.UserID.String(), token.TokenHash)
	// TTL user:tokens set = refresh token TTL (rolling update)
	pipe.Expire(ctx, userTokensPrefix+token.UserID.String(), ttl)

	_, err = pipe.Exec(ctx)
	if err != nil {
		return fmt.Errorf("save refresh token: %w", err)
	}

	return nil
}

func (r *TokenRepository) GetRefreshToken(ctx context.Context, tokenHash string) (*domain.RefreshToken, error) {
	key := refreshPrefix + tokenHash

	payload, err := r.rdb.Get(ctx, key).Bytes()
	if err != nil {
		if errors.Is(err, redis.Nil) {
			return nil, domain.ErrInvalidToken
		}
		return nil, fmt.Errorf("get refresh token: %w", err)
	}

	var data refreshTokenData
	if err := json.Unmarshal(payload, &data); err != nil {
		return nil, fmt.Errorf("unmarshal refresh token: %w", err)
	}

	userID, err := uuid.Parse(data.UserID)
	if err != nil {
		return nil, fmt.Errorf("parse user id: %w", err)
	}

	return &domain.RefreshToken{
		UserID:    userID,
		TokenHash: data.TokenHash,
		ExpiresAt: data.ExpiresAt,
	}, nil
}

func (r *TokenRepository) RevokeRefreshToken(ctx context.Context, tokenHash string) error {
	token, err := r.GetRefreshToken(ctx, tokenHash)
	if err != nil {
		if errors.Is(err, domain.ErrInvalidToken) {
			return nil // sudah revoked / expired, oke
		}
		return err
	}

	pipe := r.rdb.Pipeline()
	pipe.Del(ctx, refreshPrefix+tokenHash)
	pipe.SRem(ctx, userTokensPrefix+token.UserID.String(), tokenHash)

	_, err = pipe.Exec(ctx)
	if err != nil {
		return fmt.Errorf("revoke refresh token: %w", err)
	}

	return nil
}

func (r *TokenRepository) RevokeAllUserTokens(ctx context.Context, userID uuid.UUID) error {
	userTokensKey := userTokensPrefix + userID.String()

	tokenHashes, err := r.rdb.SMembers(ctx, userTokensKey).Result()
	if err != nil {
		return fmt.Errorf("get user tokens: %w", err)
	}

	if len(tokenHashes) == 0 {
		return nil
	}

	pipe := r.rdb.Pipeline()
	for _, hash := range tokenHashes {
		pipe.Del(ctx, refreshPrefix+hash)
	}
	pipe.Del(ctx, userTokensKey)

	_, err = pipe.Exec(ctx)
	if err != nil {
		return fmt.Errorf("revoke all user tokens: %w", err)
	}

	return nil
}

// =============================================================
// ACCESS TOKEN BLACKLIST
// TTL dinamis berdasarkan sisa waktu expire token (dari claims.ExpiresAt)
// Ini lebih aman daripada hardcode 16m karena mengikuti actual expiry
// =============================================================

type blacklistData struct {
	UserID    string    `json:"user_id"`
	Email     string    `json:"email"`
	ExpiresAt time.Time `json:"expires_at"`
}

// BlacklistAccessToken menyimpan token ke blacklist dengan TTL = sisa waktu expire + buffer clock skew 30s
// Parameter claims harus sudah include ExpiresAt dari JWT payload
func (r *TokenRepository) BlacklistAccessToken(ctx context.Context, tokenHash string, claims *domain.Claims) error {
	data := blacklistData{
		UserID:    claims.UserID,
		Email:     claims.Email,
		ExpiresAt: claims.ExpiresAt,
	}

	payload, err := json.Marshal(data)
	if err != nil {
		return fmt.Errorf("marshal blacklist data: %w", err)
	}

	// TTL = sisa waktu expire + 30 detik buffer untuk clock skew antar node
	ttl := time.Until(claims.ExpiresAt) + 30*time.Second
	if ttl <= 30*time.Second {
		// Token sudah expired, tidak perlu di-blacklist
		return nil
	}

	key := blacklistPrefix + tokenHash
	if err := r.rdb.Set(ctx, key, payload, ttl).Err(); err != nil {
		return fmt.Errorf("blacklist access token: %w", err)
	}

	return nil
}

func (r *TokenRepository) IsAccessTokenBlacklisted(ctx context.Context, tokenHash string) (bool, error) {
	key := blacklistPrefix + tokenHash

	exists, err := r.rdb.Exists(ctx, key).Result()
	if err != nil {
		return false, fmt.Errorf("check blacklist: %w", err)
	}

	return exists > 0, nil
}

// =============================================================
// Rate Limiting — disimpan di Redis dengan sliding window
// =============================================================

// IncrRateLimit menambah counter rate limit untuk key tertentu
// Mengembalikan (count saat ini, waktu reset, error)
func (r *TokenRepository) IncrRateLimit(ctx context.Context, key string, window time.Duration) (int64, time.Time, error) {
	pipe := r.rdb.Pipeline()
	incrCmd := pipe.Incr(ctx, "ratelimit:"+key)
	pipe.Expire(ctx, "ratelimit:"+key, window)

	_, err := pipe.Exec(ctx)
	if err != nil {
		return 0, time.Time{}, fmt.Errorf("rate limit incr: %w", err)
	}

	count := incrCmd.Val()
	resetAt := time.Now().Add(window)
	return count, resetAt, nil
}

// GetRateLimit mengambil counter rate limit saat ini tanpa increment
func (r *TokenRepository) GetRateLimit(ctx context.Context, key string) (int64, error) {
	val, err := r.rdb.Get(ctx, "ratelimit:"+key).Int64()
	if err != nil {
		if errors.Is(err, redis.Nil) {
			return 0, nil
		}
		return 0, fmt.Errorf("get rate limit: %w", err)
	}
	return val, nil
}

// ResetRateLimit menghapus counter (dipakai setelah login berhasil)
func (r *TokenRepository) ResetRateLimit(ctx context.Context, key string) error {
	return r.rdb.Del(ctx, "ratelimit:"+key).Err()
}

// =============================================================
// Helper
// =============================================================

// HashToken — SHA256 hash dari raw token string
// Tidak pernah simpan raw token ke storage
func HashToken(token string) string {
	hash := sha256.Sum256([]byte(token))
	return fmt.Sprintf("%x", hash)
}
