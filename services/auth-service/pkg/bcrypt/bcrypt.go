package bcrypt

import (
	"github.com/AsepNurdinDev/belanjayuk/services/auth-service/internal/domain"
	"golang.org/x/crypto/bcrypt"
)

// defaultCost — cost bcrypt default (12 lebih aman dari library default 10)
// Bisa di-override via WithCost()
const defaultCost = 12

// Hasher — wrapper bcrypt dengan configurable cost
type Hasher struct {
	cost int
}

func New(cost int) *Hasher {
	if cost < bcrypt.MinCost || cost > bcrypt.MaxCost {
		cost = defaultCost
	}
	return &Hasher{cost: cost}
}

func (h *Hasher) Hash(password string) (string, error) {
	hashed, err := bcrypt.GenerateFromPassword([]byte(password), h.cost)
	if err != nil {
		return "", err
	}
	return string(hashed), nil
}

func (h *Hasher) Verify(hashedPassword, plainPassword string) error {
	err := bcrypt.CompareHashAndPassword([]byte(hashedPassword), []byte(plainPassword))
	if err != nil {
		return domain.ErrInvalidCredentials
	}
	return nil
}

// =============================================================
// Package-level functions — untuk kemudahan penggunaan tanpa struct
// Menggunakan defaultCost = 12
// =============================================================

// HashPassword — hash password sebelum disimpan ke DB
func HashPassword(password string) (string, error) {
	hashed, err := bcrypt.GenerateFromPassword([]byte(password), defaultCost)
	if err != nil {
		return "", err
	}
	return string(hashed), nil
}

// VerifyPassword — bandingkan plain password dengan hash di DB
func VerifyPassword(hashedPassword, plainPassword string) error {
	err := bcrypt.CompareHashAndPassword([]byte(hashedPassword), []byte(plainPassword))
	if err != nil {
		return domain.ErrInvalidCredentials
	}
	return nil
}
