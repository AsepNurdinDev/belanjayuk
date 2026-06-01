package middleware

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/AsepNurdinDev/belanjayuk/services/auth-service/internal/domain"
	"github.com/AsepNurdinDev/belanjayuk/services/auth-service/internal/usecase"
)

const (
	ContextKeyUserID = "user_id"
	ContextKeyClaims = "claims"
)

// Auth — middleware autentikasi JWT
// Memvalidasi access token dari Authorization header: "Bearer <token>"
// Menyimpan claims ke gin.Context untuk diakses handler
func Auth(uc usecase.AuthUsecase) gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, errorResponse("authorization header required"))
			return
		}

		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
			c.AbortWithStatusJSON(http.StatusUnauthorized, errorResponse("invalid authorization header format, use: Bearer <token>"))
			return
		}

		token := strings.TrimSpace(parts[1])
		if token == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, errorResponse("token is empty"))
			return
		}

		claims, err := uc.ValidateToken(c.Request.Context(), token)
		if err != nil {
			switch err {
			case domain.ErrExpiredToken:
				c.AbortWithStatusJSON(http.StatusUnauthorized, errorResponse("token has expired"))
			case domain.ErrTokenBlacklist:
				c.AbortWithStatusJSON(http.StatusUnauthorized, errorResponse("token has been invalidated"))
			default:
				c.AbortWithStatusJSON(http.StatusUnauthorized, errorResponse("invalid token"))
			}
			return
		}

		c.Set(ContextKeyUserID, claims.UserID)
		c.Set(ContextKeyClaims, claims)
		c.Next()
	}
}

// RequireRole — middleware otorisasi berbasis role
// Harus dipasang setelah middleware Auth
func RequireRole(roles ...domain.UserRole) gin.HandlerFunc {
	roleSet := make(map[domain.UserRole]struct{}, len(roles))
	for _, r := range roles {
		roleSet[r] = struct{}{}
	}

	return func(c *gin.Context) {
		claimsRaw, exists := c.Get(ContextKeyClaims)
		if !exists {
			c.AbortWithStatusJSON(http.StatusUnauthorized, errorResponse("unauthorized"))
			return
		}

		claims, ok := claimsRaw.(*domain.Claims)
		if !ok {
			c.AbortWithStatusJSON(http.StatusInternalServerError, errorResponse("internal error"))
			return
		}

		if _, allowed := roleSet[claims.Role]; !allowed {
			c.AbortWithStatusJSON(http.StatusForbidden, errorResponse("insufficient permissions"))
			return
		}

		c.Next()
	}
}

// GetClaimsFromContext — helper untuk mengambil claims dari context
func GetClaimsFromContext(c *gin.Context) (*domain.Claims, bool) {
	raw, exists := c.Get(ContextKeyClaims)
	if !exists {
		return nil, false
	}
	claims, ok := raw.(*domain.Claims)
	return claims, ok
}
