package jwt

import (
	"errors"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"

	"github.com/AsepNurdinDev/belanjayuk/services/auth-service/internal/domain"
)

// =============================================================
// JWT Manager
// =============================================================

type Manager struct {
	accessSecret   string
	refreshSecret  string
	accessExpires  time.Duration
	refreshExpires time.Duration
}

func NewManager(
	accessSecret, refreshSecret string,
	accessExpires, refreshExpires time.Duration,
) *Manager {
	return &Manager{
		accessSecret:   accessSecret,
		refreshSecret:  refreshSecret,
		accessExpires:  accessExpires,
		refreshExpires: refreshExpires,
	}
}

// =============================================================
// Claims — payload JWT
// =============================================================

type accessClaims struct {
	UserID string          `json:"user_id"`
	Email  string          `json:"email"`
	Role   domain.UserRole `json:"role"`
	jwt.RegisteredClaims
}

type refreshClaims struct {
	UserID string `json:"user_id"`
	jwt.RegisteredClaims
}

// =============================================================
// GenerateAccessToken — expire pendek (15m default)
// =============================================================

func (m *Manager) GenerateAccessToken(user *domain.User) (string, error) {
	claims := accessClaims{
		UserID: user.ID.String(),
		Email:  user.Email,
		Role:   user.Role,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(m.accessExpires)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			ID:        uuid.NewString(), // jti — unik per token, untuk blacklist
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signed, err := token.SignedString([]byte(m.accessSecret))
	if err != nil {
		return "", fmt.Errorf("sign access token: %w", err)
	}

	return signed, nil
}

// =============================================================
// GenerateRefreshToken — expire panjang (7d default)
// =============================================================

func (m *Manager) GenerateRefreshToken(userID uuid.UUID) (string, error) {
	claims := refreshClaims{
		UserID: userID.String(),
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(m.refreshExpires)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			ID:        uuid.NewString(),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signed, err := token.SignedString([]byte(m.refreshSecret))
	if err != nil {
		return "", fmt.Errorf("sign refresh token: %w", err)
	}

	return signed, nil
}

// =============================================================
// ValidateAccessToken — parse & validasi access token
// =============================================================

func (m *Manager) ValidateAccessToken(tokenStr string) (*domain.Claims, error) {
	token, err := jwt.ParseWithClaims(tokenStr, &accessClaims{}, func(t *jwt.Token) (any, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", t.Header["alg"])
		}
		return []byte(m.accessSecret), nil
	})
	if err != nil {
		if errors.Is(err, jwt.ErrTokenExpired) {
			return nil, domain.ErrExpiredToken
		}
		return nil, domain.ErrInvalidToken
	}

	claims, ok := token.Claims.(*accessClaims)
	if !ok || !token.Valid {
		return nil, domain.ErrInvalidToken
	}

	return &domain.Claims{
		UserID: claims.UserID,
		Email:  claims.Email,
		Role:   claims.Role,
	}, nil
}

// =============================================================
// ValidateRefreshToken — parse & validasi refresh token
// Returns userID dan jti (untuk blacklist check)
// =============================================================

func (m *Manager) ValidateRefreshToken(tokenStr string) (userID string, jti string, err error) {
	token, err := jwt.ParseWithClaims(tokenStr, &refreshClaims{}, func(t *jwt.Token) (any, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", t.Header["alg"])
		}
		return []byte(m.refreshSecret), nil
	})
	if err != nil {
		if errors.Is(err, jwt.ErrTokenExpired) {
			return "", "", domain.ErrExpiredToken
		}
		return "", "", domain.ErrInvalidToken
	}

	claims, ok := token.Claims.(*refreshClaims)
	if !ok || !token.Valid {
		return "", "", domain.ErrInvalidToken
	}

	return claims.UserID, claims.ID, nil
}

// =============================================================
// ExtractJTI — ambil JTI dari access token tanpa validasi expire
// Dipakai saat logout untuk blacklist token yang mungkin sudah expire
// =============================================================

func (m *Manager) ExtractJTI(tokenStr string) (string, error) {
	parser := jwt.NewParser(jwt.WithoutClaimsValidation())
	token, _, err := parser.ParseUnverified(tokenStr, &accessClaims{})
	if err != nil {
		return "", domain.ErrInvalidToken
	}

	claims, ok := token.Claims.(*accessClaims)
	if !ok {
		return "", domain.ErrInvalidToken
	}

	return claims.ID, nil
}

func (m *Manager) RefreshExpires() time.Duration { return m.refreshExpires }