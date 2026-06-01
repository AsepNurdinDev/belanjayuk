package http

import (
	"errors"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/AsepNurdinDev/belanjayuk/services/auth-service/internal/domain"
	"github.com/AsepNurdinDev/belanjayuk/services/auth-service/internal/middleware"
	"github.com/AsepNurdinDev/belanjayuk/services/auth-service/internal/usecase"
	"github.com/AsepNurdinDev/belanjayuk/services/auth-service/pkg/oauth"
)

// Handler — HTTP handler untuk auth-service
type Handler struct {
	uc           usecase.AuthUsecase
	googleClient *oauth.GoogleClient // untuk generate redirect URL
}

func NewHandler(uc usecase.AuthUsecase, googleClient *oauth.GoogleClient) *Handler {
	return &Handler{uc: uc, googleClient: googleClient}
}

// =============================================================
// Auth Handlers
// =============================================================

// POST /api/v1/auth/register
func (h *Handler) Register(c *gin.Context) {
	var req usecase.RegisterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, errorResp("invalid_request", err.Error()))
		return
	}

	req.Email = strings.ToLower(strings.TrimSpace(req.Email))
	req.FullName = strings.TrimSpace(req.FullName)

	tokenPair, err := h.uc.Register(c.Request.Context(), req)
	if err != nil {
		h.handleError(c, err)
		return
	}

	c.JSON(http.StatusCreated, successResp("user registered successfully", tokenPair))
}

// POST /api/v1/auth/login
func (h *Handler) Login(c *gin.Context) {
	var req usecase.LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, errorResp("invalid_request", err.Error()))
		return
	}

	req.Email = strings.ToLower(strings.TrimSpace(req.Email))

	tokenPair, err := h.uc.Login(c.Request.Context(), req)
	if err != nil {
		h.handleError(c, err)
		return
	}

	c.JSON(http.StatusOK, successResp("login successful", tokenPair))
}

// POST /api/v1/auth/refresh
func (h *Handler) RefreshToken(c *gin.Context) {
	var body struct {
		RefreshToken string `json:"refresh_token" binding:"required"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, errorResp("invalid_request", "refresh_token is required"))
		return
	}

	tokenPair, err := h.uc.RefreshToken(c.Request.Context(), body.RefreshToken)
	if err != nil {
		h.handleError(c, err)
		return
	}

	c.JSON(http.StatusOK, successResp("token refreshed", tokenPair))
}

// POST /api/v1/auth/logout
func (h *Handler) Logout(c *gin.Context) {
	claims, ok := middleware.GetClaimsFromContext(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, errorResp("unauthorized", "missing auth claims"))
		return
	}

	authHeader := c.GetHeader("Authorization")
	parts := strings.SplitN(authHeader, " ", 2)
	if len(parts) != 2 {
		c.JSON(http.StatusBadRequest, errorResp("invalid_request", "invalid authorization header"))
		return
	}
	accessToken := parts[1]

	userID, err := uuid.Parse(claims.UserID)
	if err != nil {
		c.JSON(http.StatusBadRequest, errorResp("invalid_request", "invalid user id"))
		return
	}

	if err := h.uc.Logout(c.Request.Context(), accessToken, userID); err != nil {
		h.handleError(c, err)
		return
	}

	c.JSON(http.StatusOK, successResp("logged out successfully", nil))
}

// GET /api/v1/auth/google/login
// Redirect ke Google OAuth consent screen
func (h *Handler) GoogleLogin(c *gin.Context) {
	if h.googleClient == nil {
		c.JSON(http.StatusServiceUnavailable, errorResp("oauth_not_configured", "Google OAuth is not configured"))
		return
	}

	authURL := h.googleClient.GetAuthURL()
	c.Redirect(http.StatusTemporaryRedirect, authURL)
}

// GET /api/v1/auth/google/callback
func (h *Handler) GoogleCallback(c *gin.Context) {
	code := c.Query("code")
	state := c.Query("state")

	if code == "" || state == "" {
		c.JSON(http.StatusBadRequest, errorResp("invalid_request", "missing code or state parameter"))
		return
	}

	tokenPair, err := h.uc.LoginWithGoogle(c.Request.Context(), code, state)
	if err != nil {
		h.handleError(c, err)
		return
	}

	c.JSON(http.StatusOK, successResp("google login successful", tokenPair))
}

// =============================================================
// Profile Handlers
// =============================================================

// GET /api/v1/profile
func (h *Handler) GetProfile(c *gin.Context) {
	claims, ok := middleware.GetClaimsFromContext(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, errorResp("unauthorized", "missing auth claims"))
		return
	}

	userID, err := uuid.Parse(claims.UserID)
	if err != nil {
		c.JSON(http.StatusBadRequest, errorResp("invalid_request", "invalid user id"))
		return
	}

	profile, err := h.uc.GetProfile(c.Request.Context(), userID)
	if err != nil {
		h.handleError(c, err)
		return
	}

	// Hapus sensitive field sebelum return
	if profile.User != nil {
		profile.User.PasswordHash = nil
		profile.User.GoogleID = nil
	}

	c.JSON(http.StatusOK, successResp("", profile))
}

// PATCH /api/v1/profile
func (h *Handler) UpdateProfile(c *gin.Context) {
	claims, ok := middleware.GetClaimsFromContext(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, errorResp("unauthorized", "missing auth claims"))
		return
	}

	var req usecase.UpdateProfileRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, errorResp("invalid_request", err.Error()))
		return
	}

	userID, err := uuid.Parse(claims.UserID)
	if err != nil {
		c.JSON(http.StatusBadRequest, errorResp("invalid_request", "invalid user id"))
		return
	}

	if err := h.uc.UpdateProfile(c.Request.Context(), userID, req); err != nil {
		h.handleError(c, err)
		return
	}

	c.JSON(http.StatusOK, successResp("profile updated", nil))
}

// =============================================================
// Address Handlers
// =============================================================

// GET /api/v1/addresses
func (h *Handler) GetAddresses(c *gin.Context) {
	claims, ok := middleware.GetClaimsFromContext(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, errorResp("unauthorized", "missing auth claims"))
		return
	}

	userID, err := uuid.Parse(claims.UserID)
	if err != nil {
		c.JSON(http.StatusBadRequest, errorResp("invalid_request", "invalid user id"))
		return
	}

	addresses, err := h.uc.GetAddresses(c.Request.Context(), userID)
	if err != nil {
		h.handleError(c, err)
		return
	}

	c.JSON(http.StatusOK, successResp("", gin.H{"addresses": addresses}))
}

// POST /api/v1/addresses
func (h *Handler) CreateAddress(c *gin.Context) {
	claims, ok := middleware.GetClaimsFromContext(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, errorResp("unauthorized", "missing auth claims"))
		return
	}

	var req usecase.AddressRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, errorResp("invalid_request", err.Error()))
		return
	}

	userID, err := uuid.Parse(claims.UserID)
	if err != nil {
		c.JSON(http.StatusBadRequest, errorResp("invalid_request", "invalid user id"))
		return
	}

	address, err := h.uc.CreateAddress(c.Request.Context(), userID, req)
	if err != nil {
		h.handleError(c, err)
		return
	}

	c.JSON(http.StatusCreated, successResp("address created", address))
}

// PUT /api/v1/addresses/:id
func (h *Handler) UpdateAddress(c *gin.Context) {
	claims, ok := middleware.GetClaimsFromContext(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, errorResp("unauthorized", "missing auth claims"))
		return
	}

	addressID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, errorResp("invalid_request", "invalid address id"))
		return
	}

	var req usecase.AddressRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, errorResp("invalid_request", err.Error()))
		return
	}

	userID, err := uuid.Parse(claims.UserID)
	if err != nil {
		c.JSON(http.StatusBadRequest, errorResp("invalid_request", "invalid user id"))
		return
	}

	if err := h.uc.UpdateAddress(c.Request.Context(), addressID, userID, req); err != nil {
		h.handleError(c, err)
		return
	}

	c.JSON(http.StatusOK, successResp("address updated", nil))
}

// DELETE /api/v1/addresses/:id
func (h *Handler) DeleteAddress(c *gin.Context) {
	claims, ok := middleware.GetClaimsFromContext(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, errorResp("unauthorized", "missing auth claims"))
		return
	}

	addressID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, errorResp("invalid_request", "invalid address id"))
		return
	}

	userID, err := uuid.Parse(claims.UserID)
	if err != nil {
		c.JSON(http.StatusBadRequest, errorResp("invalid_request", "invalid user id"))
		return
	}

	if err := h.uc.DeleteAddress(c.Request.Context(), addressID, userID); err != nil {
		h.handleError(c, err)
		return
	}

	c.JSON(http.StatusOK, successResp("address deleted", nil))
}

// =============================================================
// Health Check
// =============================================================

// GET /health
func (h *Handler) Health(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"status":  "ok",
		"service": "auth-service",
	})
}

// =============================================================
// Error Handling
// =============================================================

func (h *Handler) handleError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, domain.ErrUserNotFound):
		c.JSON(http.StatusNotFound, errorResp("not_found", err.Error()))
	case errors.Is(err, domain.ErrUserAlreadyExists):
		c.JSON(http.StatusConflict, errorResp("conflict", err.Error()))
	case errors.Is(err, domain.ErrInvalidCredentials):
		// Jangan bedakan antara email tidak ada vs password salah (mencegah user enumeration)
		c.JSON(http.StatusUnauthorized, errorResp("invalid_credentials", "invalid email or password"))
	case errors.Is(err, domain.ErrUserNotVerified):
		c.JSON(http.StatusForbidden, errorResp("email_not_verified", err.Error()))
	case errors.Is(err, domain.ErrInvalidToken),
		errors.Is(err, domain.ErrExpiredToken),
		errors.Is(err, domain.ErrRevokedToken),
		errors.Is(err, domain.ErrTokenBlacklist):
		c.JSON(http.StatusUnauthorized, errorResp("invalid_token", err.Error()))
	case errors.Is(err, domain.ErrForbidden):
		c.JSON(http.StatusForbidden, errorResp("forbidden", err.Error()))
	case errors.Is(err, domain.ErrAddressNotFound):
		c.JSON(http.StatusNotFound, errorResp("not_found", err.Error()))
	case errors.Is(err, domain.ErrAddressLimitExceed):
		c.JSON(http.StatusUnprocessableEntity, errorResp("limit_exceeded", err.Error()))
	case errors.Is(err, domain.ErrOAuthFailed):
		c.JSON(http.StatusBadGateway, errorResp("oauth_error", "OAuth authentication failed"))
	case errors.Is(err, domain.ErrInvalidOAuthState):
		c.JSON(http.StatusBadRequest, errorResp("invalid_state", err.Error()))
	case errors.Is(err, domain.ErrTooManyRequests):
		c.JSON(http.StatusTooManyRequests, errorResp("too_many_requests", err.Error()))
	default:
		c.JSON(http.StatusInternalServerError, errorResp("internal_error", "An unexpected error occurred"))
	}
}

// =============================================================
// Response Helpers
// =============================================================

func successResp(message string, data interface{}) gin.H {
	resp := gin.H{"success": true}
	if message != "" {
		resp["message"] = message
	}
	if data != nil {
		resp["data"] = data
	}
	return resp
}

func errorResp(code, message string) gin.H {
	return gin.H{
		"success": false,
		"error":   code,
		"message": message,
	}
}
