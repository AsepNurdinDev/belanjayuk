package domain

import "errors"

// =============================================================
// Domain Errors
// Didefinisikan di domain agar usecase & delivery bisa handle
// tanpa import package lain
// =============================================================

var (
	// User errors
	ErrUserNotFound       = errors.New("user not found")
	ErrUserAlreadyExists  = errors.New("email already registered")
	ErrInvalidCredentials = errors.New("invalid email or password")
	ErrUserNotVerified    = errors.New("email not verified")

	// Token errors
	ErrInvalidToken   = errors.New("invalid token")
	ErrExpiredToken   = errors.New("token has expired")
	ErrRevokedToken   = errors.New("token has been revoked")
	ErrTokenBlacklist = errors.New("token has been invalidated")

	// OAuth errors
	ErrOAuthFailed        = errors.New("oauth authentication failed")
	ErrGoogleIDNotFound   = errors.New("google account not linked")

	// Address errors
	ErrAddressNotFound    = errors.New("address not found")
	ErrAddressLimitExceed = errors.New("maximum 10 addresses allowed")

	// General
	ErrUnauthorized = errors.New("unauthorized")
	ErrForbidden    = errors.New("forbidden")
)