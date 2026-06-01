package middleware

import (
	"fmt"
	"math/rand"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/AsepNurdinDev/belanjayuk/services/auth-service/internal/config"
)

// SecurityHeaders — menambahkan security headers ke setiap response
// Mengikuti OWASP best practices untuk microservice
func SecurityHeaders() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Header("X-Content-Type-Options", "nosniff")
		c.Header("X-Frame-Options", "DENY")
		c.Header("X-XSS-Protection", "1; mode=block")
		c.Header("Strict-Transport-Security", "max-age=31536000; includeSubDomains")
		c.Header("Referrer-Policy", "strict-origin-when-cross-origin")
		c.Header("Content-Security-Policy", "default-src 'none'; frame-ancestors 'none'")
		c.Header("Server", "")
		c.Header("X-Powered-By", "")
		c.Next()
	}
}

// CORS — middleware CORS yang aman
// Hanya mengizinkan origin yang terdaftar di config
func CORS(cfg config.SecurityConfig) gin.HandlerFunc {
	allowedOrigins := make(map[string]struct{}, len(cfg.AllowedOrigins))
	for _, o := range cfg.AllowedOrigins {
		allowedOrigins[o] = struct{}{}
	}

	return func(c *gin.Context) {
		origin := c.GetHeader("Origin")

		if origin != "" {
			if _, allowed := allowedOrigins[origin]; allowed {
				c.Header("Access-Control-Allow-Origin", origin)
				c.Header("Vary", "Origin")
			}
			// Origin tidak diizinkan = tidak set header, browser akan block
		}

		c.Header("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, OPTIONS")
		c.Header("Access-Control-Allow-Headers", "Authorization, Content-Type, X-Request-ID")
		c.Header("Access-Control-Max-Age", "3600")

		if c.Request.Method == http.MethodOptions {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}

		c.Next()
	}
}

// MaxBodySize — membatasi ukuran request body
// Mencegah DoS via large payload (1MB default untuk auth endpoint)
func MaxBodySize(maxBytes int64) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, maxBytes)
		c.Next()
	}
}

// RequestID — menambahkan X-Request-ID ke setiap request untuk distributed tracing
func RequestID() gin.HandlerFunc {
	rng := rand.New(rand.NewSource(time.Now().UnixNano()))
	return func(c *gin.Context) {
		requestID := c.GetHeader("X-Request-ID")
		if requestID == "" {
			requestID = fmt.Sprintf("%d-%d", time.Now().UnixNano(), rng.Int63())
		}
		c.Set("request_id", requestID)
		c.Header("X-Request-ID", requestID)
		c.Next()
	}
}

// RecoveryWithJSON — mengganti gin default recovery agar return JSON bukan HTML
// Mencegah stack trace leak ke client
func RecoveryWithJSON() gin.HandlerFunc {
	return gin.CustomRecovery(func(c *gin.Context, recovered interface{}) {
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{
			"error":   "internal_server_error",
			"message": "An unexpected error occurred",
		})
	})
}

// errorResponse — helper format error response yang konsisten di semua middleware
func errorResponse(msg string) gin.H {
	return gin.H{
		"error":   "error",
		"message": msg,
	}
}
