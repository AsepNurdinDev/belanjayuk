package bcrypt

import (
	"github.com/AsepNurdinDev/belanjayuk/services/auth-service/internal/domain"
	"golang.org/x/crypto/bcrypt"
)

const defaultCost = bcrypt.DefaultCost // 10

// HashPassword — hash password sebelum disimpan ke DB
func HashPassword(password string) (string, error) {
	hashed, err := bcrypt.GenerateFromPassword([]byte(password), defaultCost)
	if err != nil {
		return "", err
	}
	return string(hashed), nil
}

// VerifyPassword — bandingkan plain password dengan hash di DB
// Returns domain.ErrInvalidCredentials kalau tidak cocok
func VerifyPassword(hashedPassword, plainPassword string) error {
	err := bcrypt.CompareHashAndPassword([]byte(hashedPassword), []byte(plainPassword))
	if err != nil {
		return domain.ErrInvalidCredentials
	}
	return nil
}