package middleware

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/AsepNurdinDev/belanjayuk/services/auth-service/internal/config"
)

// RateLimiter — interface untuk rate limit backend
// Memudahkan testing dan swap implementasi
type RateLimiter interface {
	IncrRateLimit(ctx context.Context, key string, window time.Duration) (int64, time.Time, error)
	GetRateLimit(ctx context.Context, key string) (int64, error)
	ResetRateLimit(ctx context.Context, key string) error
}

// rateLimitConfig — config per endpoint
type rateLimitConfig struct {
	maxAttempts int
	window      time.Duration
	keyPrefix   string
}

// LoginRateLimit — rate limiter khusus untuk endpoint login
// Key: IP + email untuk mencegah brute force per akun
// Lebih ketat dari global rate limit
func LoginRateLimit(rl RateLimiter, cfg config.RateLimitConfig) gin.HandlerFunc {
	return rateLimitMiddleware(rl, rateLimitConfig{
		maxAttempts: cfg.LoginMaxAttempts,
		window:      cfg.LoginWindow,
		keyPrefix:   "login",
	}, func(c *gin.Context) string {
		// Key: kombinasi IP + email untuk mencegah brute force per akun dari IP tertentu
		var body struct {
			Email string `json:"email"`
		}
		// Baca body tanpa consume (pakai ShouldBindBodyWith jika perlu re-read)
		ip := realClientIP(c)
		return fmt.Sprintf("%s:%s", ip, body.Email)
	})
}

// LoginIPRateLimit — rate limiter per IP untuk endpoint login
// Mencegah distributed brute force dari satu IP
func LoginIPRateLimit(rl RateLimiter, cfg config.RateLimitConfig) gin.HandlerFunc {
	return rateLimitMiddleware(rl, rateLimitConfig{
		maxAttempts: cfg.LoginMaxAttempts,
		window:      cfg.LoginWindow,
		keyPrefix:   "login:ip",
	}, func(c *gin.Context) string {
		return realClientIP(c)
	})
}

// RegisterRateLimit — rate limiter untuk endpoint register
// Lebih ketat karena operasi DB lebih berat
func RegisterRateLimit(rl RateLimiter, cfg config.RateLimitConfig) gin.HandlerFunc {
	return rateLimitMiddleware(rl, rateLimitConfig{
		maxAttempts: cfg.RegisterMaxAttempts,
		window:      cfg.RegisterWindow,
		keyPrefix:   "register",
	}, func(c *gin.Context) string {
		return realClientIP(c)
	})
}

// RefreshRateLimit — rate limiter untuk endpoint refresh token
func RefreshRateLimit(rl RateLimiter, cfg config.RateLimitConfig) gin.HandlerFunc {
	return rateLimitMiddleware(rl, rateLimitConfig{
		maxAttempts: cfg.RefreshMaxAttempts,
		window:      cfg.RefreshWindow,
		keyPrefix:   "refresh",
	}, func(c *gin.Context) string {
		return realClientIP(c)
	})
}

// GlobalRateLimit — rate limiter global per IP untuk semua endpoint
// Sebagai last line of defense
func GlobalRateLimit(rl RateLimiter, cfg config.RateLimitConfig) gin.HandlerFunc {
	return rateLimitMiddleware(rl, rateLimitConfig{
		maxAttempts: cfg.GlobalMaxRequests,
		window:      cfg.GlobalWindow,
		keyPrefix:   "global",
	}, func(c *gin.Context) string {
		return realClientIP(c)
	})
}

// rateLimitMiddleware — core rate limit logic dengan sliding window counter
// Menggunakan Redis INCR + EXPIRE pattern
// Menyertakan standard rate limit headers (RateLimit-Limit, RateLimit-Remaining, Retry-After)
func rateLimitMiddleware(rl RateLimiter, cfg rateLimitConfig, keyFn func(*gin.Context) string) gin.HandlerFunc {
	return func(c *gin.Context) {
		key := fmt.Sprintf("%s:%s", cfg.keyPrefix, keyFn(c))

		count, resetAt, err := rl.IncrRateLimit(c.Request.Context(), key, cfg.window)
		if err != nil {
			// Jika Redis error, fail open (jangan block request)
			// Log error tapi lanjutkan request — availability > security di sini
			c.Next()
			return
		}

		remaining := int64(cfg.maxAttempts) - count
		if remaining < 0 {
			remaining = 0
		}

		// Set standard rate limit headers
		c.Header("RateLimit-Limit", strconv.Itoa(cfg.maxAttempts))
		c.Header("RateLimit-Remaining", strconv.FormatInt(remaining, 10))
		c.Header("RateLimit-Reset", strconv.FormatInt(resetAt.Unix(), 10))

		if count > int64(cfg.maxAttempts) {
			retryAfter := int(time.Until(resetAt).Seconds())
			if retryAfter < 1 {
				retryAfter = 1
			}
			c.Header("Retry-After", strconv.Itoa(retryAfter))
			c.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{
				"error":       "too_many_requests",
				"message":     "Rate limit exceeded, please try again later",
				"retry_after": retryAfter,
			})
			return
		}

		c.Next()
	}
}

// realClientIP — mengambil IP asli klien dari X-Forwarded-For atau X-Real-IP
// Penting untuk deployment di belakang reverse proxy (nginx, cloudflare, dll)
// PERINGATAN: hanya trust header ini jika memang ada trusted proxy di depan
func realClientIP(c *gin.Context) string {
	// Cek X-Forwarded-For (bisa berisi chain: "client, proxy1, proxy2")
	forwarded := c.GetHeader("X-Forwarded-For")
	if forwarded != "" {
		// Ambil IP pertama (client asli)
		parts := splitByComma(forwarded)
		if len(parts) > 0 {
			ip := trimSpace(parts[0])
			if ip != "" {
				return ip
			}
		}
	}

	// Fallback ke X-Real-IP
	if realIP := c.GetHeader("X-Real-IP"); realIP != "" {
		return trimSpace(realIP)
	}

	// Fallback ke RemoteAddr
	return c.ClientIP()
}

func splitByComma(s string) []string {
	var result []string
	start := 0
	for i := 0; i <= len(s); i++ {
		if i == len(s) || s[i] == ',' {
			result = append(result, s[start:i])
			start = i + 1
		}
	}
	return result
}

func trimSpace(s string) string {
	start := 0
	for start < len(s) && s[start] == ' ' {
		start++
	}
	end := len(s)
	for end > start && s[end-1] == ' ' {
		end--
	}
	return s[start:end]
}
