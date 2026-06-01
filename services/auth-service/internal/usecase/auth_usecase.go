package usecase

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/rs/zerolog/log"

	"github.com/AsepNurdinDev/belanjayuk/services/auth-service/internal/domain"
	"github.com/AsepNurdinDev/belanjayuk/services/auth-service/pkg/bcrypt"
	redisrepo "github.com/AsepNurdinDev/belanjayuk/services/auth-service/internal/repository/redis"
)

// =============================================================
// Register — daftar dengan email & password
// =============================================================

func (u *authUsecase) Register(ctx context.Context, req RegisterRequest) (*domain.TokenPair, error) {
	// Cek email sudah terdaftar atau belum
	_, err := u.userRepo.GetByEmail(ctx, req.Email)
	if err == nil {
		return nil, domain.ErrUserAlreadyExists
	}
	if err != domain.ErrUserNotFound {
		return nil, fmt.Errorf("check email: %w", err)
	}

	hashedPassword, err := bcrypt.HashPassword(req.Password)
	if err != nil {
		return nil, fmt.Errorf("hash password: %w", err)
	}

	now := time.Now()
	user := &domain.User{
		ID:           uuid.New(),
		Email:        req.Email,
		PasswordHash: &hashedPassword,
		FullName:     req.FullName,
		Role:         domain.RoleCustomer,
		AuthMethod:   domain.AuthMethodLocal,
		IsVerified:   false,
		CreatedAt:    now,
		UpdatedAt:    now,
	}
	if req.Phone != "" {
		user.Phone = &req.Phone
	}

	if err := u.userRepo.Create(ctx, user); err != nil {
		return nil, err
	}

	// Buat profile kosong
	profile := &domain.UserProfile{
		ID:        uuid.New(),
		UserID:    user.ID,
		CreatedAt: now,
		UpdatedAt: now,
	}
	if err := u.userRepo.CreateProfile(ctx, profile); err != nil {
		return nil, fmt.Errorf("create profile: %w", err)
	}

	// Publish event — non-blocking, jangan gagalkan register karena event error
	if err := u.publisher.PublishUserRegistered(ctx, user); err != nil {
		log.Warn().Err(err).Str("user_id", user.ID.String()).Msg("failed to publish user.registered event")
	}

	return u.generateTokenPair(ctx, user)
}

// =============================================================
// Login — login dengan email & password
// =============================================================

func (u *authUsecase) Login(ctx context.Context, req LoginRequest) (*domain.TokenPair, error) {
	user, err := u.userRepo.GetByEmail(ctx, req.Email)
	if err != nil {
		if err == domain.ErrUserNotFound {
			// Timing attack mitigation: lakukan operasi bcrypt dummy
			_ = bcrypt.VerifyPassword("$2a$12$dummy_hash_to_prevent_timing_attack_padding__", req.Password)
			return nil, domain.ErrInvalidCredentials
		}
		return nil, fmt.Errorf("get user: %w", err)
	}

	// Google user tidak bisa login dengan password
	if user.IsGoogle() || user.PasswordHash == nil {
		_ = bcrypt.VerifyPassword("$2a$12$dummy_hash_to_prevent_timing_attack_padding__", req.Password)
		return nil, domain.ErrInvalidCredentials
	}

	if err := bcrypt.VerifyPassword(*user.PasswordHash, req.Password); err != nil {
		return nil, domain.ErrInvalidCredentials
	}

	pair, err := u.generateTokenPair(ctx, user)
	if err != nil {
		return nil, err
	}

	// Publish event login (untuk audit log / analytics)
	if err := u.publisher.PublishUserLoggedIn(ctx, user, ""); err != nil {
		log.Warn().Err(err).Str("user_id", user.ID.String()).Msg("failed to publish user.logged_in event")
	}

	return pair, nil
}

// =============================================================
// LoginWithGoogle — login/register via Google OAuth
// =============================================================

func (u *authUsecase) LoginWithGoogle(ctx context.Context, code, state string) (*domain.TokenPair, error) {
	if err := u.google.VerifyState(state); err != nil {
		return nil, domain.ErrInvalidOAuthState
	}

	token, err := u.google.ExchangeCode(ctx, code)
	if err != nil {
		return nil, domain.ErrOAuthFailed
	}

	googleUser, err := u.google.GetUserInfo(ctx, token)
	if err != nil {
		return nil, domain.ErrOAuthFailed
	}

	// Cek apakah user sudah pernah login dengan Google ini
	user, err := u.userRepo.GetByGoogleID(ctx, googleUser.ID)
	if err == nil {
		return u.generateTokenPair(ctx, user)
	}
	if err != domain.ErrUserNotFound {
		return nil, fmt.Errorf("get user by google id: %w", err)
	}

	// Cek apakah email sudah terdaftar (local user)
	existingUser, err := u.userRepo.GetByEmail(ctx, googleUser.Email)
	if err == nil {
		existingUser.GoogleID = &googleUser.ID
		existingUser.IsVerified = true
		existingUser.UpdatedAt = time.Now()
		if err := u.userRepo.Update(ctx, existingUser); err != nil {
			return nil, fmt.Errorf("link google account: %w", err)
		}
		return u.generateTokenPair(ctx, existingUser)
	}
	if err != domain.ErrUserNotFound {
		return nil, fmt.Errorf("check email: %w", err)
	}

	// User baru — register otomatis via Google
	now := time.Now()
	newUser := &domain.User{
		ID:         uuid.New(),
		Email:      googleUser.Email,
		FullName:   googleUser.FullName,
		GoogleID:   &googleUser.ID,
		Role:       domain.RoleCustomer,
		AuthMethod: domain.AuthMethodGoogle,
		IsVerified: true,
		CreatedAt:  now,
		UpdatedAt:  now,
	}

	if err := u.userRepo.Create(ctx, newUser); err != nil {
		return nil, err
	}

	profile := &domain.UserProfile{
		ID:        uuid.New(),
		UserID:    newUser.ID,
		CreatedAt: now,
		UpdatedAt: now,
	}
	if googleUser.Picture != "" {
		profile.AvatarURL = &googleUser.Picture
	}
	if err := u.userRepo.CreateProfile(ctx, profile); err != nil {
		return nil, fmt.Errorf("create profile: %w", err)
	}

	if err := u.publisher.PublishUserRegistered(ctx, newUser); err != nil {
		log.Warn().Err(err).Str("user_id", newUser.ID.String()).Msg("failed to publish user.registered event (google)")
	}

	return u.generateTokenPair(ctx, newUser)
}

// =============================================================
// RefreshToken — rotate refresh token, issue access token baru
// =============================================================

func (u *authUsecase) RefreshToken(ctx context.Context, refreshToken string) (*domain.TokenPair, error) {
	userIDStr, _, err := u.jwtMgr.ValidateRefreshToken(refreshToken)
	if err != nil {
		return nil, err
	}

	tokenHash := redisrepo.HashToken(refreshToken)
	storedToken, err := u.tokenRepo.GetRefreshToken(ctx, tokenHash)
	if err != nil {
		return nil, domain.ErrInvalidToken
	}

	if !storedToken.IsValid() {
		return nil, domain.ErrRevokedToken
	}

	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		return nil, domain.ErrInvalidToken
	}

	user, err := u.userRepo.GetByID(ctx, userID)
	if err != nil {
		return nil, err
	}

	// Revoke refresh token lama SEBELUM issue yang baru (token rotation)
	if err := u.tokenRepo.RevokeRefreshToken(ctx, tokenHash); err != nil {
		return nil, fmt.Errorf("revoke old token: %w", err)
	}

	return u.generateTokenPair(ctx, user)
}

// =============================================================
// Logout — blacklist access token + revoke semua refresh token
// =============================================================

func (u *authUsecase) Logout(ctx context.Context, accessToken string, userID uuid.UUID) error {
	claims, err := u.jwtMgr.ValidateAccessToken(accessToken)
	if err != nil && err != domain.ErrExpiredToken {
		return domain.ErrInvalidToken
	}

	if claims != nil {
		tokenHash := redisrepo.HashToken(accessToken)
		if err := u.tokenRepo.BlacklistAccessToken(ctx, tokenHash, claims); err != nil {
			return fmt.Errorf("blacklist token: %w", err)
		}
	}

	if err := u.tokenRepo.RevokeAllUserTokens(ctx, userID); err != nil {
		return fmt.Errorf("revoke tokens: %w", err)
	}

	if err := u.publisher.PublishUserLoggedOut(ctx, userID.String()); err != nil {
		log.Warn().Err(err).Str("user_id", userID.String()).Msg("failed to publish user.logged_out event")
	}

	return nil
}

// =============================================================
// GetProfile
// =============================================================

func (u *authUsecase) GetProfile(ctx context.Context, userID uuid.UUID) (*ProfileResponse, error) {
	user, err := u.userRepo.GetByID(ctx, userID)
	if err != nil {
		return nil, err
	}

	profile, err := u.userRepo.GetProfileByUserID(ctx, userID)
	if err != nil {
		return nil, err
	}

	return &ProfileResponse{
		User:    user,
		Profile: profile,
	}, nil
}

// =============================================================
// UpdateProfile
// =============================================================

func (u *authUsecase) UpdateProfile(ctx context.Context, userID uuid.UUID, req UpdateProfileRequest) error {
	user, err := u.userRepo.GetByID(ctx, userID)
	if err != nil {
		return err
	}

	profile, err := u.userRepo.GetProfileByUserID(ctx, userID)
	if err != nil {
		return err
	}

	if req.FullName != "" {
		user.FullName = req.FullName
	}
	if req.Phone != "" {
		user.Phone = &req.Phone
	}
	user.UpdatedAt = time.Now()

	if req.Bio != "" {
		profile.Bio = &req.Bio
	}
	if req.AvatarURL != "" {
		profile.AvatarURL = &req.AvatarURL
	}
	if req.Gender != "" {
		g := domain.Gender(req.Gender)
		profile.Gender = &g
	}
	if req.BirthDate != "" {
		t, err := time.Parse("2006-01-02", req.BirthDate)
		if err != nil {
			return fmt.Errorf("invalid birth_date format, use YYYY-MM-DD: %w", err)
		}
		profile.BirthDate = &t
	}
	profile.UpdatedAt = time.Now()

	if err := u.userRepo.Update(ctx, user); err != nil {
		return err
	}

	return u.userRepo.UpdateProfile(ctx, profile)
}

// =============================================================
// Address
// =============================================================

func (u *authUsecase) GetAddresses(ctx context.Context, userID uuid.UUID) ([]*domain.UserAddress, error) {
	return u.userRepo.GetAddressesByUserID(ctx, userID)
}

func (u *authUsecase) CreateAddress(ctx context.Context, userID uuid.UUID, req AddressRequest) (*domain.UserAddress, error) {
	if req.IsDefault {
		if err := u.userRepo.UnsetDefaultAddress(ctx, userID); err != nil {
			return nil, fmt.Errorf("unset default: %w", err)
		}
	}

	now := time.Now()
	address := &domain.UserAddress{
		ID:         uuid.New(),
		UserID:     userID,
		Label:      req.Label,
		Recipient:  req.Recipient,
		Phone:      req.Phone,
		Province:   req.Province,
		City:       req.City,
		District:   req.District,
		PostalCode: req.PostalCode,
		Detail:     req.Detail,
		IsDefault:  req.IsDefault,
		CreatedAt:  now,
		UpdatedAt:  now,
	}

	if err := u.userRepo.CreateAddress(ctx, address); err != nil {
		return nil, err
	}

	return address, nil
}

func (u *authUsecase) UpdateAddress(ctx context.Context, id uuid.UUID, userID uuid.UUID, req AddressRequest) error {
	address, err := u.userRepo.GetAddressByID(ctx, id)
	if err != nil {
		return err
	}

	if address.UserID != userID {
		return domain.ErrForbidden
	}

	if req.IsDefault && !address.IsDefault {
		if err := u.userRepo.UnsetDefaultAddress(ctx, userID); err != nil {
			return fmt.Errorf("unset default: %w", err)
		}
	}

	address.Label = req.Label
	address.Recipient = req.Recipient
	address.Phone = req.Phone
	address.Province = req.Province
	address.City = req.City
	address.District = req.District
	address.PostalCode = req.PostalCode
	address.Detail = req.Detail
	address.IsDefault = req.IsDefault
	address.UpdatedAt = time.Now()

	return u.userRepo.UpdateAddress(ctx, address)
}

func (u *authUsecase) DeleteAddress(ctx context.Context, id uuid.UUID, userID uuid.UUID) error {
	address, err := u.userRepo.GetAddressByID(ctx, id)
	if err != nil {
		return err
	}
	if address.UserID != userID {
		return domain.ErrForbidden
	}
	return u.userRepo.DeleteAddress(ctx, id, userID)
}

// =============================================================
// ValidateToken — dipanggil gRPC server
// =============================================================

func (u *authUsecase) ValidateToken(ctx context.Context, accessToken string) (*domain.Claims, error) {
	claims, err := u.jwtMgr.ValidateAccessToken(accessToken)
	if err != nil {
		return nil, err
	}

	tokenHash := redisrepo.HashToken(accessToken)
	blacklisted, err := u.tokenRepo.IsAccessTokenBlacklisted(ctx, tokenHash)
	if err != nil {
		return nil, fmt.Errorf("check blacklist: %w", err)
	}
	if blacklisted {
		return nil, domain.ErrTokenBlacklist
	}

	return claims, nil
}

// =============================================================
// generateTokenPair — helper buat access + refresh token
// =============================================================

func (u *authUsecase) generateTokenPair(ctx context.Context, user *domain.User) (*domain.TokenPair, error) {
	accessToken, err := u.jwtMgr.GenerateAccessToken(user)
	if err != nil {
		return nil, fmt.Errorf("generate access token: %w", err)
	}

	rawRefreshToken, err := u.jwtMgr.GenerateRefreshToken(user.ID)
	if err != nil {
		return nil, fmt.Errorf("generate refresh token: %w", err)
	}

	refreshTokenHash := redisrepo.HashToken(rawRefreshToken)
	refreshToken := &domain.RefreshToken{
		ID:        uuid.New(),
		UserID:    user.ID,
		TokenHash: refreshTokenHash,
		ExpiresAt: time.Now().Add(u.jwtMgr.RefreshExpires()),
		CreatedAt: time.Now(),
	}
	if err := u.tokenRepo.SaveRefreshToken(ctx, refreshToken); err != nil {
		return nil, fmt.Errorf("save refresh token: %w", err)
	}

	return &domain.TokenPair{
		AccessToken:  accessToken,
		RefreshToken: rawRefreshToken,
	}, nil
}
