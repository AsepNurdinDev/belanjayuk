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
    App       AppConfig
    DB        DBConfig
    Redis     RedisConfig
    JWT       JWTConfig
    Google    GoogleConfig
    RabbitMQ  RabbitMQConfig
    RateLimit RateLimitConfig
    Security  SecurityConfig
}

type AppConfig struct {
    Env      string // development | staging | production
    Port     string
    GRPCPort string
}

type DBConfig struct {
    Host            string
    Port            string
    Name            string
    User            string
    Password        string
    SSLMode         string
    MaxOpenConns    int
    MaxIdleConns    int
    ConnMaxLifetime time.Duration
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
    // TLS
    TLSEnabled bool
}

func (r RedisConfig) Addr() string {
    return fmt.Sprintf("%s:%s", r.Host, r.Port)
}

type JWTConfig struct {
    AccessSecret   string
    RefreshSecret  string
    AccessExpires  time.Duration
    RefreshExpires time.Duration
    Issuer         string
}

type GoogleConfig struct {
    ClientID     string
    ClientSecret string
    RedirectURL  string
}

type RabbitMQConfig struct {
    URL string
}

// RateLimitConfig — konfigurasi rate limiting per endpoint
type RateLimitConfig struct {
    // Login: mencegah brute force
    LoginMaxAttempts int
    LoginWindow      time.Duration
    // Register: mencegah spam akun
    RegisterMaxAttempts int
    RegisterWindow      time.Duration
    // Global per IP
    GlobalMaxRequests int
    GlobalWindow      time.Duration
    // Token refresh
    RefreshMaxAttempts int
    RefreshWindow      time.Duration
}

// SecurityConfig — security headers dan CORS
type SecurityConfig struct {
    AllowedOrigins   []string
    TrustedProxies   []string
    BCryptCost       int
    OAuthStateSecret string // untuk CSRF protection pada OAuth
}

// =============================================================
// Load — baca config dari environment variables
// =============================================================

func Load() (*Config, error) {
    viper.AutomaticEnv()

    // App defaults
    viper.SetDefault("APP_ENV", "development")
    viper.SetDefault("APP_PORT", "8001")
    viper.SetDefault("APP_GRPC_PORT", "9001")

    // DB defaults
    viper.SetDefault("DB_HOST", "localhost")
    viper.SetDefault("DB_PORT", "5433")
    viper.SetDefault("DB_NAME", "auth_db")
    viper.SetDefault("DB_USER", "postgres")
    viper.SetDefault("DB_SSL_MODE", "disable")
    viper.SetDefault("DB_MAX_OPEN_CONNS", 25)
    viper.SetDefault("DB_MAX_IDLE_CONNS", 5)
    viper.SetDefault("DB_CONN_MAX_LIFETIME", "5m")

    // Redis defaults
    viper.SetDefault("REDIS_HOST", "localhost")
    viper.SetDefault("REDIS_PORT", "6379")
    viper.SetDefault("REDIS_DB", 0)
    viper.SetDefault("REDIS_TLS_ENABLED", false)

    // JWT defaults
    viper.SetDefault("JWT_ACCESS_EXPIRES", "15m")
    viper.SetDefault("JWT_REFRESH_EXPIRES", "168h") // 7 days
    viper.SetDefault("JWT_ISSUER", "belanjayuk-auth-service")

    // Rate limit defaults (conservative for production)
    viper.SetDefault("RATE_LIMIT_LOGIN_MAX", 5)
    viper.SetDefault("RATE_LIMIT_LOGIN_WINDOW", "15m")
    viper.SetDefault("RATE_LIMIT_REGISTER_MAX", 3)
    viper.SetDefault("RATE_LIMIT_REGISTER_WINDOW", "1h")
    viper.SetDefault("RATE_LIMIT_GLOBAL_MAX", 100)
    viper.SetDefault("RATE_LIMIT_GLOBAL_WINDOW", "1m")
    viper.SetDefault("RATE_LIMIT_REFRESH_MAX", 10)
    viper.SetDefault("RATE_LIMIT_REFRESH_WINDOW", "15m")

    // Security defaults
    viper.SetDefault("BCRYPT_COST", 12)

    // =============================================================
    // PERBAIKAN: Deteksi Fleksibel untuk Monorepo / Workspaces
    // =============================================================
    viper.SetConfigName(".env")
    viper.SetConfigType("env")
    
    // Jalur 1: Jika dijalankan dari dalam folder services/auth-service
    viper.AddConfigPath(".") 
    
    // Jalur 2: Jika dijalankan dari root workspace /belanjayuk
    viper.AddConfigPath("./services/auth-service") 
    
    _ = viper.ReadInConfig() 

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

    // Validasi secret length (minimal 32 karakter untuk keamanan)
    if len(viper.GetString("JWT_ACCESS_SECRET")) < 32 {
        return nil, fmt.Errorf("JWT_ACCESS_SECRET must be at least 32 characters")
    }
    if len(viper.GetString("JWT_REFRESH_SECRET")) < 32 {
        return nil, fmt.Errorf("JWT_REFRESH_SECRET must be at least 32 characters")
    }

    accessExpires, err := time.ParseDuration(viper.GetString("JWT_ACCESS_EXPIRES"))
    if err != nil {
        return nil, fmt.Errorf("invalid JWT_ACCESS_EXPIRES: %w", err)
    }

    refreshExpires, err := time.ParseDuration(viper.GetString("JWT_REFRESH_EXPIRES"))
    if err != nil {
        return nil, fmt.Errorf("invalid JWT_REFRESH_EXPIRES: %w", err)
    }

    connMaxLifetime, err := time.ParseDuration(viper.GetString("DB_CONN_MAX_LIFETIME"))
    if err != nil {
        return nil, fmt.Errorf("invalid DB_CONN_MAX_LIFETIME: %w", err)
    }

    loginWindow, err := time.ParseDuration(viper.GetString("RATE_LIMIT_LOGIN_WINDOW"))
    if err != nil {
        return nil, fmt.Errorf("invalid RATE_LIMIT_LOGIN_WINDOW: %w", err)
    }
    registerWindow, err := time.ParseDuration(viper.GetString("RATE_LIMIT_REGISTER_WINDOW"))
    if err != nil {
        return nil, fmt.Errorf("invalid RATE_LIMIT_REGISTER_WINDOW: %w", err)
    }
    globalWindow, err := time.ParseDuration(viper.GetString("RATE_LIMIT_GLOBAL_WINDOW"))
    if err != nil {
        return nil, fmt.Errorf("invalid RATE_LIMIT_GLOBAL_WINDOW: %w", err)
    }
    refreshWindow, err := time.ParseDuration(viper.GetString("RATE_LIMIT_REFRESH_WINDOW"))
    if err != nil {
        return nil, fmt.Errorf("invalid RATE_LIMIT_REFRESH_WINDOW: %w", err)
    }

    // Parse allowed origins (comma-separated)
    allowedOriginsRaw := viper.GetString("CORS_ALLOWED_ORIGINS")
    var allowedOrigins []string
    if allowedOriginsRaw != "" {
        for _, o := range splitTrim(allowedOriginsRaw, ",") {
            allowedOrigins = append(allowedOrigins, o)
        }
    }

    return &Config{
        App: AppConfig{
            Env:      viper.GetString("APP_ENV"),
            Port:     viper.GetString("APP_PORT"),
            GRPCPort: viper.GetString("APP_GRPC_PORT"),
        },
        DB: DBConfig{
            Host:            viper.GetString("DB_HOST"),
            Port:            viper.GetString("DB_PORT"),
            Name:            viper.GetString("DB_NAME"),
            User:            viper.GetString("DB_USER"),
            Password:        viper.GetString("DB_PASSWORD"),
            SSLMode:         viper.GetString("DB_SSL_MODE"),
            MaxOpenConns:    viper.GetInt("DB_MAX_OPEN_CONNS"),
            MaxIdleConns:    viper.GetInt("DB_MAX_IDLE_CONNS"),
            ConnMaxLifetime: connMaxLifetime,
        },
        Redis: RedisConfig{
            Host:       viper.GetString("REDIS_HOST"),
            Port:       viper.GetString("REDIS_PORT"),
            Password:   viper.GetString("REDIS_PASSWORD"),
            DB:         viper.GetInt("REDIS_DB"),
            TLSEnabled: viper.GetBool("REDIS_TLS_ENABLED"),
        },
        JWT: JWTConfig{
            AccessSecret:   viper.GetString("JWT_ACCESS_SECRET"),
            RefreshSecret:  viper.GetString("JWT_REFRESH_SECRET"),
            AccessExpires:  accessExpires,
            RefreshExpires: refreshExpires,
            Issuer:         viper.GetString("JWT_ISSUER"),
        },
        Google: GoogleConfig{
            ClientID:     viper.GetString("GOOGLE_CLIENT_ID"),
            ClientSecret: viper.GetString("GOOGLE_CLIENT_SECRET"),
            RedirectURL:  viper.GetString("GOOGLE_REDIRECT_URL"),
        },
        RabbitMQ: RabbitMQConfig{
            URL: viper.GetString("RABBITMQ_URL"),
        },
        RateLimit: RateLimitConfig{
            LoginMaxAttempts:    viper.GetInt("RATE_LIMIT_LOGIN_MAX"),
            LoginWindow:         loginWindow,
            RegisterMaxAttempts: viper.GetInt("RATE_LIMIT_REGISTER_MAX"),
            RegisterWindow:      registerWindow,
            GlobalMaxRequests:   viper.GetInt("RATE_LIMIT_GLOBAL_MAX"),
            GlobalWindow:        globalWindow,
            RefreshMaxAttempts:  viper.GetInt("RATE_LIMIT_REFRESH_MAX"),
            RefreshWindow:       refreshWindow,
        },
        Security: SecurityConfig{
            AllowedOrigins:   allowedOrigins,
            BCryptCost:       viper.GetInt("BCRYPT_COST"),
            OAuthStateSecret: viper.GetString("OAUTH_STATE_SECRET"),
        },
    }, nil
}

func (c *Config) IsDevelopment() bool {
    return c.App.Env == "development"
}

func (c *Config) IsProduction() bool {
    return c.App.Env == "production"
}

// splitTrim memisah string dan trim whitespace tiap element
func splitTrim(s, sep string) []string {
    var result []string
    for i := 0; i < len(s); {
        j := len(s)
        for k := i; k < len(s); k++ {
            if s[k] == sep[0] {
                j = k
                break
            }
        }
        part := trim(s[i:j])
        if part != "" {
            result = append(result, part)
        }
        i = j + 1
    }
    return result
}

func trim(s string) string {
    start := 0
    for start < len(s) && (s[start] == ' ' || s[start] == '\t') {
        start++
    }
    end := len(s)
    for end > start && (s[end-1] == ' ' || s[end-1] == '\t') {
        end--
    }
    return s[start:end]
}