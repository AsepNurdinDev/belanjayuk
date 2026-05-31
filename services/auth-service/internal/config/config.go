package config

import (
	"fmt"
	"time"

	"github.com/spf13/viper"
)

// =============================================================
// Config — semua konfigurasi auth-service
// Di-load dari environment variables atau file .env
// =============================================================

type Config struct {
	App      AppConfig
	DB       DBConfig
	Redis    RedisConfig
	JWT      JWTConfig
	Google   GoogleConfig
	RabbitMQ RabbitMQConfig
}

type AppConfig struct {
	Env  string // development | staging | production
	Port string
	GRPCPort string
}

type DBConfig struct {
	Host     string
	Port     string
	Name     string
	User     string
	Password string
	SSLMode  string
}

func (d DBConfig) DSN() string {
	return fmt.Sprintf(
		"host=%s port=%s dbname=%s user=%s password=%s sslmode=%s",
		d.Host, d.Port, d.Name, d.User, d.Password, d.SSLMode,
	)
}

type RedisConfig struct {
	Host     string
	Port     string
	Password string
	DB       int
}

func (r RedisConfig) Addr() string {
	return fmt.Sprintf("%s:%s", r.Host, r.Port)
}

type JWTConfig struct {
	AccessSecret   string
	RefreshSecret  string
	AccessExpires  time.Duration
	RefreshExpires time.Duration
}

type GoogleConfig struct {
	ClientID     string
	ClientSecret string
	RedirectURL  string
}

type RabbitMQConfig struct {
	URL string
}

// =============================================================
// Load — baca config dari environment variables
// =============================================================

func Load() (*Config, error) {
	viper.AutomaticEnv()

	// Default values
	viper.SetDefault("APP_ENV", "development")
	viper.SetDefault("APP_PORT", "8001")
	viper.SetDefault("APP_GRPC_PORT", "9001")
	viper.SetDefault("DB_HOST", "localhost")
	viper.SetDefault("DB_PORT", "5432")
	viper.SetDefault("DB_NAME", "auth_db")
	viper.SetDefault("DB_USER", "postgres")
	viper.SetDefault("DB_SSL_MODE", "disable")
	viper.SetDefault("REDIS_HOST", "localhost")
	viper.SetDefault("REDIS_PORT", "6379")
	viper.SetDefault("REDIS_DB", 0)
	viper.SetDefault("JWT_ACCESS_EXPIRES", "15m")
	viper.SetDefault("JWT_REFRESH_EXPIRES", "168h") // 7 days

	// Baca .env kalau ada (development only)
	viper.SetConfigFile(".env")
	viper.SetConfigType("env")
	_ = viper.ReadInConfig() // ignore error kalau .env tidak ada (production pakai env vars)

	// Validasi required fields
	required := []string{
		"DB_PASSWORD",
		"JWT_ACCESS_SECRET",
		"JWT_REFRESH_SECRET",
	}
	for _, key := range required {
		if viper.GetString(key) == "" {
			return nil, fmt.Errorf("required env var %s is not set", key)
		}
	}

	accessExpires, err := time.ParseDuration(viper.GetString("JWT_ACCESS_EXPIRES"))
	if err != nil {
		return nil, fmt.Errorf("invalid JWT_ACCESS_EXPIRES: %w", err)
	}

	refreshExpires, err := time.ParseDuration(viper.GetString("JWT_REFRESH_EXPIRES"))
	if err != nil {
		return nil, fmt.Errorf("invalid JWT_REFRESH_EXPIRES: %w", err)
	}

	return &Config{
		App: AppConfig{
			Env:      viper.GetString("APP_ENV"),
			Port:     viper.GetString("APP_PORT"),
			GRPCPort: viper.GetString("APP_GRPC_PORT"),
		},
		DB: DBConfig{
			Host:     viper.GetString("DB_HOST"),
			Port:     viper.GetString("DB_PORT"),
			Name:     viper.GetString("DB_NAME"),
			User:     viper.GetString("DB_USER"),
			Password: viper.GetString("DB_PASSWORD"),
			SSLMode:  viper.GetString("DB_SSL_MODE"),
		},
		Redis: RedisConfig{
			Host:     viper.GetString("REDIS_HOST"),
			Port:     viper.GetString("REDIS_PORT"),
			Password: viper.GetString("REDIS_PASSWORD"),
			DB:       viper.GetInt("REDIS_DB"),
		},
		JWT: JWTConfig{
			AccessSecret:   viper.GetString("JWT_ACCESS_SECRET"),
			RefreshSecret:  viper.GetString("JWT_REFRESH_SECRET"),
			AccessExpires:  accessExpires,
			RefreshExpires: refreshExpires,
		},
		Google: GoogleConfig{
			ClientID:     viper.GetString("GOOGLE_CLIENT_ID"),
			ClientSecret: viper.GetString("GOOGLE_CLIENT_SECRET"),
			RedirectURL:  viper.GetString("GOOGLE_REDIRECT_URL"),
		},
		RabbitMQ: RabbitMQConfig{
			URL: viper.GetString("RABBITMQ_URL"),
		},
	}, nil
}

func (c *Config) IsDevelopment() bool {
	return c.App.Env == "development"
}

func (c *Config) IsProduction() bool {
	return c.App.Env == "production"
}