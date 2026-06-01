package domain

import "errors"

// =============================================================
// Domain Errors — sentinel errors untuk seluruh auth-service
// Digunakan oleh usecase dan dipetakan ke HTTP/gRPC status code
// oleh delivery layer
// =============================================================

var (
	// User errors
	ErrUserNotFound      = errors.New("user not found")
	ErrUserAlreadyExists = errors.New("email already registered")
	ErrUserNotVerified   = errors.New("email not verified, please check your inbox")

	// Credential / auth errors
	ErrInvalidCredentials = errors.New("invalid email or password")
	ErrUnauthorized       = errors.New("unauthorized")
	ErrForbidden          = errors.New("forbidden: insufficient permissions")

	// Token errors
	ErrInvalidToken   = errors.New("invalid token")
	ErrExpiredToken   = errors.New("token has expired")
	ErrRevokedToken   = errors.New("token has been revoked")
	ErrTokenBlacklist = errors.New("token has been invalidated")

	// OAuth errors
	ErrOAuthFailed      = errors.New("oauth authentication failed")
	ErrInvalidOAuthState = errors.New("invalid oauth state, possible CSRF attack")

	// Address errors
	ErrAddressNotFound   = errors.New("address not found")
	ErrAddressLimitExceed = errors.New("maximum 10 addresses per user")

	// Rate limit
	ErrTooManyRequests = errors.New("too many requests, please try again later")
)
