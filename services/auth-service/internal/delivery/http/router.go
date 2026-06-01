package http

import (
	"github.com/gin-gonic/gin"

	"github.com/AsepNurdinDev/belanjayuk/services/auth-service/internal/config"
	"github.com/AsepNurdinDev/belanjayuk/services/auth-service/internal/domain"
	"github.com/AsepNurdinDev/belanjayuk/services/auth-service/internal/middleware"
	"github.com/AsepNurdinDev/belanjayuk/services/auth-service/internal/usecase"
	"github.com/AsepNurdinDev/belanjayuk/services/auth-service/pkg/oauth"
)

const (
	maxBodyBytes int64 = 1 * 1024 * 1024 // 1 MB
)

// NewRouter — setup Gin router dengan semua middleware dan routes
func NewRouter(
	cfg *config.Config,
	uc usecase.AuthUsecase,
	rateLimiter middleware.RateLimiter,
) *gin.Engine {
	if cfg.IsProduction() {
		gin.SetMode(gin.ReleaseMode)
	}

	// Buat engine tanpa default middleware (kita pasang sendiri yang lebih aman)
	r := gin.New()

	// =========================================================
	// Global Middleware (urutan penting!)
	// =========================================================
	r.Use(middleware.RecoveryWithJSON())       // Recover dari panic, return JSON
	r.Use(middleware.RequestID())              // Tambah X-Request-ID untuk tracing
	r.Use(middleware.SecurityHeaders())        // OWASP security headers
	r.Use(middleware.CORS(cfg.Security))       // CORS dengan allowed origins
	r.Use(middleware.MaxBodySize(maxBodyBytes)) // Limit request body 1MB

	// Global rate limit per IP (100 req/menit default)
	r.Use(middleware.GlobalRateLimit(rateLimiter, cfg.RateLimit))

	// Setup Google OAuth client untuk handler
	var googleClient *oauth.GoogleClient
	if cfg.Google.ClientID != "" {
		googleClient = oauth.NewGoogleClient(
			cfg.Google.ClientID,
			cfg.Google.ClientSecret,
			cfg.Google.RedirectURL,
		).WithStateSecret(cfg.Security.OAuthStateSecret)
	}

	handler := NewHandler(uc, googleClient)

	// =========================================================
	// Health Check — tanpa auth, tanpa rate limit ketat
	// =========================================================
	r.GET("/health", handler.Health)
	r.GET("/ready", handler.Health) // kubernetes readiness probe

	// =========================================================
	// API v1
	// =========================================================
	api := r.Group("/api/v1")

	// Auth endpoints (public)
	auth := api.Group("/auth")
	{
		// Register — rate limit ketat (3x/jam per IP)
		auth.POST("/register",
			middleware.RegisterRateLimit(rateLimiter, cfg.RateLimit),
			handler.Register,
		)

		// Login — rate limit sangat ketat (5x/15 menit per IP)
		auth.POST("/login",
			middleware.LoginIPRateLimit(rateLimiter, cfg.RateLimit),
			handler.Login,
		)

		// Refresh token — rate limit sedang
		auth.POST("/refresh",
			middleware.RefreshRateLimit(rateLimiter, cfg.RateLimit),
			handler.RefreshToken,
		)

		// Logout — butuh auth
		auth.POST("/logout",
			middleware.Auth(uc),
			handler.Logout,
		)

		// Google OAuth
		auth.GET("/google/login", handler.GoogleLogin)
		auth.GET("/google/callback", handler.GoogleCallback)
	}

	// Profile endpoints (protected)
	profile := api.Group("/profile", middleware.Auth(uc))
	{
		profile.GET("", handler.GetProfile)
		profile.PATCH("", handler.UpdateProfile)
	}

	// Address endpoints (protected)
	addresses := api.Group("/addresses", middleware.Auth(uc))
	{
		addresses.GET("", handler.GetAddresses)
		addresses.POST("", handler.CreateAddress)
		addresses.PUT("/:id", handler.UpdateAddress)
		addresses.DELETE("/:id", handler.DeleteAddress)
	}

	// Admin endpoints (protected + role check)
	admin := api.Group("/admin",
		middleware.Auth(uc),
		middleware.RequireRole(domain.RoleAdmin, domain.RoleSuperAdmin),
	)
	{
		// TODO: tambahkan admin endpoints di sini
		_ = admin
	}

	return r
}
