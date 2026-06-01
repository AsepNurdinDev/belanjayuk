package postgres

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/AsepNurdinDev/belanjayuk/services/auth-service/internal/domain"
)

// =============================================================
// UserRepository — implementasi domain.UserRepository
// =============================================================

type UserRepository struct {
	db *pgxpool.Pool
}

func NewUserRepository(db *pgxpool.Pool) *UserRepository {
	return &UserRepository{db: db}
}

// =============================================================
// USER
// =============================================================

func (r *UserRepository) Create(ctx context.Context, user *domain.User) error {
	query := `
		INSERT INTO users (id, email, password_hash, full_name, phone, role, auth_method, google_id, is_verified, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
	`
	_, err := r.db.Exec(ctx, query,
		user.ID,
		user.Email,
		user.PasswordHash,
		user.FullName,
		user.Phone,
		user.Role,
		user.AuthMethod,
		user.GoogleID,
		user.IsVerified,
		user.CreatedAt,
		user.UpdatedAt,
	)
	if err != nil {
		if isUniqueViolation(err) {
			return domain.ErrUserAlreadyExists
		}
		return fmt.Errorf("create user: %w", err)
	}
	return nil
}

func (r *UserRepository) GetByID(ctx context.Context, id uuid.UUID) (*domain.User, error) {
	query := `
		SELECT id, email, password_hash, full_name, phone, role, auth_method, google_id, is_verified, created_at, updated_at
		FROM users WHERE id = $1
	`
	user, err := scanUser(r.db.QueryRow(ctx, query, id))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrUserNotFound
		}
		return nil, fmt.Errorf("get user by id: %w", err)
	}
	return user, nil
}

func (r *UserRepository) GetByEmail(ctx context.Context, email string) (*domain.User, error) {
	query := `
		SELECT id, email, password_hash, full_name, phone, role, auth_method, google_id, is_verified, created_at, updated_at
		FROM users WHERE email = $1
	`
	user, err := scanUser(r.db.QueryRow(ctx, query, email))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrUserNotFound
		}
		return nil, fmt.Errorf("get user by email: %w", err)
	}
	return user, nil
}

func (r *UserRepository) GetByGoogleID(ctx context.Context, googleID string) (*domain.User, error) {
	query := `
		SELECT id, email, password_hash, full_name, phone, role, auth_method, google_id, is_verified, created_at, updated_at
		FROM users WHERE google_id = $1
	`
	user, err := scanUser(r.db.QueryRow(ctx, query, googleID))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrUserNotFound
		}
		return nil, fmt.Errorf("get user by google id: %w", err)
	}
	return user, nil
}

func (r *UserRepository) Update(ctx context.Context, user *domain.User) error {
	query := `
		UPDATE users
		SET email = $1, full_name = $2, phone = $3, is_verified = $4, updated_at = $5
		WHERE id = $6
	`
	user.UpdatedAt = time.Now()
	result, err := r.db.Exec(ctx, query,
		user.Email,
		user.FullName,
		user.Phone,
		user.IsVerified,
		user.UpdatedAt,
		user.ID,
	)
	if err != nil {
		return fmt.Errorf("update user: %w", err)
	}
	if result.RowsAffected() == 0 {
		return domain.ErrUserNotFound
	}
	return nil
}

// =============================================================
// PROFILE
// =============================================================

func (r *UserRepository) CreateProfile(ctx context.Context, profile *domain.UserProfile) error {
	query := `
		INSERT INTO user_profiles (id, user_id, avatar_url, bio, birth_date, gender, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
	`
	_, err := r.db.Exec(ctx, query,
		profile.ID,
		profile.UserID,
		profile.AvatarURL,
		profile.Bio,
		profile.BirthDate,
		profile.Gender,
		profile.CreatedAt,
		profile.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("create profile: %w", err)
	}
	return nil
}

func (r *UserRepository) GetProfileByUserID(ctx context.Context, userID uuid.UUID) (*domain.UserProfile, error) {
	query := `
		SELECT id, user_id, avatar_url, bio, birth_date, gender, created_at, updated_at
		FROM user_profiles WHERE user_id = $1
	`
	row := r.db.QueryRow(ctx, query, userID)
	profile := &domain.UserProfile{}
	err := row.Scan(
		&profile.ID,
		&profile.UserID,
		&profile.AvatarURL,
		&profile.Bio,
		&profile.BirthDate,
		&profile.Gender,
		&profile.CreatedAt,
		&profile.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrUserNotFound
		}
		return nil, fmt.Errorf("get profile: %w", err)
	}
	return profile, nil
}

func (r *UserRepository) UpdateProfile(ctx context.Context, profile *domain.UserProfile) error {
	query := `
		UPDATE user_profiles
		SET avatar_url = $1, bio = $2, birth_date = $3, gender = $4, updated_at = $5
		WHERE user_id = $6
	`
	profile.UpdatedAt = time.Now()
	result, err := r.db.Exec(ctx, query,
		profile.AvatarURL,
		profile.Bio,
		profile.BirthDate,
		profile.Gender,
		profile.UpdatedAt,
		profile.UserID,
	)
	if err != nil {
		return fmt.Errorf("update profile: %w", err)
	}
	if result.RowsAffected() == 0 {
		return domain.ErrUserNotFound
	}
	return nil
}

// =============================================================
// ADDRESS
// =============================================================

func (r *UserRepository) CreateAddress(ctx context.Context, address *domain.UserAddress) error {
	// Batasi maksimal 10 alamat per user
	var count int
	err := r.db.QueryRow(ctx, `SELECT COUNT(*) FROM user_addresses WHERE user_id = $1`, address.UserID).Scan(&count)
	if err != nil {
		return fmt.Errorf("count addresses: %w", err)
	}
	if count >= 10 {
		return domain.ErrAddressLimitExceed
	}

	query := `
		INSERT INTO user_addresses (id, user_id, label, recipient, phone, province, city, district, postal_code, detail, is_default, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13)
	`
	_, err = r.db.Exec(ctx, query,
		address.ID,
		address.UserID,
		address.Label,
		address.Recipient,
		address.Phone,
		address.Province,
		address.City,
		address.District,
		address.PostalCode,
		address.Detail,
		address.IsDefault,
		address.CreatedAt,
		address.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("create address: %w", err)
	}
	return nil
}

func (r *UserRepository) GetAddressByID(ctx context.Context, id uuid.UUID) (*domain.UserAddress, error) {
	query := `
		SELECT id, user_id, label, recipient, phone, province, city, district, postal_code, detail, is_default, created_at, updated_at
		FROM user_addresses WHERE id = $1
	`
	address, err := scanAddress(r.db.QueryRow(ctx, query, id))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrAddressNotFound
		}
		return nil, fmt.Errorf("get address by id: %w", err)
	}
	return address, nil
}

func (r *UserRepository) GetAddressesByUserID(ctx context.Context, userID uuid.UUID) ([]*domain.UserAddress, error) {
	query := `
		SELECT id, user_id, label, recipient, phone, province, city, district, postal_code, detail, is_default, created_at, updated_at
		FROM user_addresses WHERE user_id = $1
		ORDER BY is_default DESC, created_at ASC
	`
	rows, err := r.db.Query(ctx, query, userID)
	if err != nil {
		return nil, fmt.Errorf("get addresses: %w", err)
	}
	defer rows.Close()

	var addresses []*domain.UserAddress
	for rows.Next() {
		address, err := scanAddress(rows)
		if err != nil {
			return nil, fmt.Errorf("scan address: %w", err)
		}
		addresses = append(addresses, address)
	}
	return addresses, nil
}

func (r *UserRepository) UpdateAddress(ctx context.Context, address *domain.UserAddress) error {
	query := `
		UPDATE user_addresses
		SET label = $1, recipient = $2, phone = $3, province = $4, city = $5,
		    district = $6, postal_code = $7, detail = $8, is_default = $9, updated_at = $10
		WHERE id = $11 AND user_id = $12
	`
	address.UpdatedAt = time.Now()
	result, err := r.db.Exec(ctx, query,
		address.Label,
		address.Recipient,
		address.Phone,
		address.Province,
		address.City,
		address.District,
		address.PostalCode,
		address.Detail,
		address.IsDefault,
		address.UpdatedAt,
		address.ID,
		address.UserID,
	)
	if err != nil {
		return fmt.Errorf("update address: %w", err)
	}
	if result.RowsAffected() == 0 {
		return domain.ErrAddressNotFound
	}
	return nil
}

func (r *UserRepository) DeleteAddress(ctx context.Context, id uuid.UUID, userID uuid.UUID) error {
	result, err := r.db.Exec(ctx,
		`DELETE FROM user_addresses WHERE id = $1 AND user_id = $2`,
		id, userID,
	)
	if err != nil {
		return fmt.Errorf("delete address: %w", err)
	}
	if result.RowsAffected() == 0 {
		return domain.ErrAddressNotFound
	}
	return nil
}

func (r *UserRepository) UnsetDefaultAddress(ctx context.Context, userID uuid.UUID) error {
	_, err := r.db.Exec(ctx,
		`UPDATE user_addresses SET is_default = false WHERE user_id = $1`,
		userID,
	)
	if err != nil {
		return fmt.Errorf("unset default address: %w", err)
	}
	return nil
}

// =============================================================
// Helpers
// =============================================================

// scanUser — reusable scanner untuk menghindari duplikasi
func scanUser(row pgx.Row) (*domain.User, error) {
	user := &domain.User{}
	err := row.Scan(
		&user.ID,
		&user.Email,
		&user.PasswordHash,
		&user.FullName,
		&user.Phone,
		&user.Role,
		&user.AuthMethod,
		&user.GoogleID,
		&user.IsVerified,
		&user.CreatedAt,
		&user.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	return user, nil
}

// scanAddress — reusable scanner untuk address
func scanAddress(row pgx.Row) (*domain.UserAddress, error) {
	address := &domain.UserAddress{}
	err := row.Scan(
		&address.ID,
		&address.UserID,
		&address.Label,
		&address.Recipient,
		&address.Phone,
		&address.Province,
		&address.City,
		&address.District,
		&address.PostalCode,
		&address.Detail,
		&address.IsDefault,
		&address.CreatedAt,
		&address.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	return address, nil
}

// isUniqueViolation — cek apakah error adalah duplicate key (PostgreSQL error code 23505)
func isUniqueViolation(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "23505"
}