package domain

import (
	"time"

	"github.com/google/uuid"
)

// =============================================================
// Enums / Constants
// =============================================================

type AuthMethod string
type UserRole string
type Gender string

const (
	AuthMethodLocal  AuthMethod = "local"
	AuthMethodGoogle AuthMethod = "google"
)

const (
	RoleCustomer   UserRole = "customer"
	RoleAdmin      UserRole = "admin"
	RoleSuperAdmin UserRole = "super_admin"
)

const (
	GenderMale   Gender = "male"
	GenderFemale Gender = "female"
	GenderOther  Gender = "other"
)

// =============================================================
// User — core authentication entity
// =============================================================

type User struct {
	ID           uuid.UUID  `db:"id"`
	Email        string     `db:"email"`
	PasswordHash *string    `db:"password_hash"`
	FullName     string     `db:"full_name"`
	Phone        *string    `db:"phone"`
	Role         UserRole   `db:"role"`
	AuthMethod   AuthMethod `db:"auth_method"`
	GoogleID     *string    `db:"google_id"`
	IsVerified   bool       `db:"is_verified"`
	CreatedAt    time.Time  `db:"created_at"`
	UpdatedAt    time.Time  `db:"updated_at"`
}

func (u *User) IsLocal() bool {
	return u.AuthMethod == AuthMethodLocal
}

func (u *User) IsGoogle() bool {
	return u.AuthMethod == AuthMethodGoogle
}

func (u *User) IsAdmin() bool {
	return u.Role == RoleAdmin || u.Role == RoleSuperAdmin
}

// =============================================================
// UserProfile — extended identity data
// =============================================================

type UserProfile struct {
	ID        uuid.UUID  `db:"id"`
	UserID    uuid.UUID  `db:"user_id"`
	AvatarURL *string    `db:"avatar_url"`
	Bio       *string    `db:"bio"`
	BirthDate *time.Time `db:"birth_date"`
	Gender    *Gender    `db:"gender"`
	CreatedAt time.Time  `db:"created_at"`
	UpdatedAt time.Time  `db:"updated_at"`
}

// =============================================================
// UserAddress — alamat pengiriman milik user
// =============================================================

type UserAddress struct {
	ID         uuid.UUID `db:"id"`
	UserID     uuid.UUID `db:"user_id"`
	Label      string    `db:"label"`
	Recipient  string    `db:"recipient"`
	Phone      string    `db:"phone"`
	Province   string    `db:"province"`
	City       string    `db:"city"`
	District   string    `db:"district"`
	PostalCode string    `db:"postal_code"`
	Detail     string    `db:"detail"`
	IsDefault  bool      `db:"is_default"`
	CreatedAt  time.Time `db:"created_at"`
	UpdatedAt  time.Time `db:"updated_at"`
}

// =============================================================
// RefreshToken
// =============================================================

type RefreshToken struct {
	ID        uuid.UUID  `db:"id"`
	UserID    uuid.UUID  `db:"user_id"`
	TokenHash string     `db:"token_hash"`
	ExpiresAt time.Time  `db:"expires_at"`
	RevokedAt *time.Time `db:"revoked_at"`
	CreatedAt time.Time  `db:"created_at"`
}

func (r *RefreshToken) IsExpired() bool {
	return time.Now().After(r.ExpiresAt)
}

func (r *RefreshToken) IsRevoked() bool {
	return r.RevokedAt != nil
}

func (r *RefreshToken) IsValid() bool {
	return !r.IsExpired() && !r.IsRevoked()
}

// =============================================================
// Token Pair — output setelah login/register berhasil
// =============================================================

type TokenPair struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
}

// =============================================================
// Claims — payload JWT access token
// ExpiresAt diperlukan untuk menghitung TTL blacklist secara dinamis
// =============================================================

type Claims struct {
	UserID    string    `json:"user_id"`
	Email     string    `json:"email"`
	Role      UserRole  `json:"role"`
	ExpiresAt time.Time `json:"expires_at"` // dari JWT "exp" claim
}
