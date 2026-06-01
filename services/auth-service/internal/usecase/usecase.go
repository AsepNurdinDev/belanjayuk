package usecase

import (
	"context"

	"github.com/google/uuid"

	"github.com/AsepNurdinDev/belanjayuk/services/auth-service/internal/domain"
	"github.com/AsepNurdinDev/belanjayuk/services/auth-service/internal/event"
	"github.com/AsepNurdinDev/belanjayuk/services/auth-service/pkg/jwt"
	"github.com/AsepNurdinDev/belanjayuk/services/auth-service/pkg/oauth"
)

// =============================================================
// AuthUsecase — kontrak business logic untuk delivery layer
// =============================================================

type AuthUsecase interface {
	// Auth
	Register(ctx context.Context, req RegisterRequest) (*domain.TokenPair, error)
	Login(ctx context.Context, req LoginRequest) (*domain.TokenPair, error)
	LoginWithGoogle(ctx context.Context, code, state string) (*domain.TokenPair, error)
	RefreshToken(ctx context.Context, refreshToken string) (*domain.TokenPair, error)
	Logout(ctx context.Context, accessToken string, userID uuid.UUID) error

	// Profile
	GetProfile(ctx context.Context, userID uuid.UUID) (*ProfileResponse, error)
	UpdateProfile(ctx context.Context, userID uuid.UUID, req UpdateProfileRequest) error

	// Address
	GetAddresses(ctx context.Context, userID uuid.UUID) ([]*domain.UserAddress, error)
	CreateAddress(ctx context.Context, userID uuid.UUID, req AddressRequest) (*domain.UserAddress, error)
	UpdateAddress(ctx context.Context, id uuid.UUID, userID uuid.UUID, req AddressRequest) error
	DeleteAddress(ctx context.Context, id uuid.UUID, userID uuid.UUID) error

	// Internal — dipanggil gRPC server
	ValidateToken(ctx context.Context, accessToken string) (*domain.Claims, error)
}

// =============================================================
// Request / Response DTOs
// =============================================================

type RegisterRequest struct {
	Email    string `json:"email"     validate:"required,email"`
	Password string `json:"password"  validate:"required,min=8"`
	FullName string `json:"full_name"  validate:"required,min=2,max=100"`
	Phone    string `json:"phone"     validate:"omitempty,min=8,max=20"`
}

type LoginRequest struct {
	Email    string `json:"email"    validate:"required,email"`
	Password string `json:"password" validate:"required"`
}

type UpdateProfileRequest struct {
	FullName  string `json:"full_name"  validate:"omitempty,min=2,max=100"`
	Phone     string `json:"phone"      validate:"omitempty,min=8,max=20"`
	Bio       string `json:"bio"        validate:"omitempty,max=500"`
	BirthDate string `json:"birth_date" validate:"omitempty"` // format: YYYY-MM-DD
	Gender    string `json:"gender"     validate:"omitempty,oneof=male female other"`
	AvatarURL string `json:"avatar_url" validate:"omitempty,url"`
}

type AddressRequest struct {
	Label      string `json:"label"       validate:"required,max=50"`
	Recipient  string `json:"recipient"   validate:"required,max=100"`
	Phone      string `json:"phone"       validate:"required,min=8,max=20"`
	Province   string `json:"province"    validate:"required"`
	City       string `json:"city"        validate:"required"`
	District   string `json:"district"    validate:"required"`
	PostalCode string `json:"postal_code" validate:"required,len=5"`
	Detail     string `json:"detail"      validate:"required,max=500"`
	IsDefault  bool   `json:"is_default"`
}

type ProfileResponse struct {
	User    *domain.User        `json:"user"`
	Profile *domain.UserProfile `json:"profile"`
}

// =============================================================
// authUsecase — implementasi AuthUsecase
// =============================================================

type authUsecase struct {
	userRepo  domain.UserRepository
	tokenRepo domain.TokenRepository
	jwtMgr    *jwt.Manager
	google    *oauth.GoogleClient
	publisher event.EventPublisher
}

// NewAuthUsecase — constructor dengan dependency injection
// publisher bisa NoOpPublisher jika RabbitMQ tidak dikonfigurasi
func NewAuthUsecase(
	userRepo domain.UserRepository,
	tokenRepo domain.TokenRepository,
	jwtMgr *jwt.Manager,
	google *oauth.GoogleClient,
	publisher event.EventPublisher,
) AuthUsecase {
	return &authUsecase{
		userRepo:  userRepo,
		tokenRepo: tokenRepo,
		jwtMgr:    jwtMgr,
		google:    google,
		publisher: publisher,
	}
}
