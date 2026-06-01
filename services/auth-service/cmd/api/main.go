package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"

	"github.com/AsepNurdinDev/belanjayuk/services/auth-service/internal/config"
	deliverygrpc "github.com/AsepNurdinDev/belanjayuk/services/auth-service/internal/delivery/grpc"
	deliveryhttp "github.com/AsepNurdinDev/belanjayuk/services/auth-service/internal/delivery/http"
	"github.com/AsepNurdinDev/belanjayuk/services/auth-service/internal/event"
	"github.com/AsepNurdinDev/belanjayuk/services/auth-service/internal/middleware"
	postgresrepo "github.com/AsepNurdinDev/belanjayuk/services/auth-service/internal/repository/postgres"
	redisrepo "github.com/AsepNurdinDev/belanjayuk/services/auth-service/internal/repository/redis"
	"github.com/AsepNurdinDev/belanjayuk/services/auth-service/internal/usecase"
	"github.com/AsepNurdinDev/belanjayuk/services/auth-service/pkg/jwt"
	"github.com/AsepNurdinDev/belanjayuk/services/auth-service/pkg/oauth"
)

func main() {
	// ==========================================================
	// Logger — zerolog dengan pretty print di development
	// ==========================================================
	log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr, TimeFormat: time.RFC3339})
	zerolog.TimeFieldFormat = zerolog.TimeFormatUnix

	// ==========================================================
	// Config
	// ==========================================================
	cfg, err := config.Load()
	if err != nil {
		log.Fatal().Err(err).Msg("failed to load config")
	}

	if cfg.IsProduction() {
		zerolog.SetGlobalLevel(zerolog.InfoLevel)
	} else {
		zerolog.SetGlobalLevel(zerolog.DebugLevel)
	}

	log.Info().
		Str("env", cfg.App.Env).
		Str("http_port", cfg.App.Port).
		Str("grpc_port", cfg.App.GRPCPort).
		Msg("starting auth-service")

	// ==========================================================
	// PostgreSQL
	// ==========================================================
	dbCtx, dbCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer dbCancel()

	poolCfg, err := pgxpool.ParseConfig(cfg.DB.DSN())
	if err != nil {
		log.Fatal().Err(err).Msg("failed to parse db config")
	}
	poolCfg.MaxConns = int32(cfg.DB.MaxOpenConns)
	poolCfg.MinConns = int32(cfg.DB.MaxIdleConns)
	poolCfg.MaxConnLifetime = cfg.DB.ConnMaxLifetime

	pool, err := pgxpool.NewWithConfig(dbCtx, poolCfg)
	if err != nil {
		log.Fatal().Err(err).Msg("failed to connect to postgres")
	}
	defer pool.Close()

	if err := pool.Ping(dbCtx); err != nil {
		log.Fatal().Err(err).Msg("postgres ping failed")
	}
	log.Info().Msg("connected to postgres")

	// ==========================================================
	// Redis
	// ==========================================================
	rdb := redis.NewClient(&redis.Options{
		Addr:     cfg.Redis.Addr(),
		Password: cfg.Redis.Password,
		DB:       cfg.Redis.DB,
	})
	defer rdb.Close()

	redisCtx, redisCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer redisCancel()

	if err := rdb.Ping(redisCtx).Err(); err != nil {
		log.Fatal().Err(err).Msg("redis ping failed")
	}
	log.Info().Msg("connected to redis")

	// ==========================================================
	// Repositories
	// ==========================================================
	userRepo := postgresrepo.NewUserRepository(pool)
	tokenRepo := redisrepo.NewTokenRepository(rdb)

	// ==========================================================
	// JWT Manager
	// ==========================================================
	jwtMgr := jwt.NewManager(
		cfg.JWT.AccessSecret,
		cfg.JWT.RefreshSecret,
		cfg.JWT.AccessExpires,
		cfg.JWT.RefreshExpires,
	)

	// ==========================================================
	// Google OAuth Client
	// ==========================================================
	googleClient := oauth.NewGoogleClient(
		cfg.Google.ClientID,
		cfg.Google.ClientSecret,
		cfg.Google.RedirectURL,
	)

	// ==========================================================
	// Event Publisher
	// ==========================================================
	var publisher event.EventPublisher
	if cfg.RabbitMQ.URL != "" {
		pub, err := event.NewPublisher(cfg.RabbitMQ.URL)
		if err != nil {
			log.Warn().Err(err).Msg("failed to connect to rabbitmq, using no-op publisher")
			publisher = &event.NoOpPublisher{}
		} else {
			publisher = pub
			defer publisher.Close()
			log.Info().Msg("connected to rabbitmq")
		}
	} else {
		log.Warn().Msg("RABBITMQ_URL not set, using no-op publisher")
		publisher = &event.NoOpPublisher{}
	}

	// ==========================================================
	// Usecase
	// ==========================================================
	authUC := usecase.NewAuthUsecase(userRepo, tokenRepo, jwtMgr, googleClient, publisher)

	// ==========================================================
	// Rate Limiter (menggunakan tokenRepo yang sudah ada Redis)
	// ==========================================================
	rateLimiter := middleware.NewRedisRateLimiter(rdb)

	// ==========================================================
	// HTTP Server
	// ==========================================================
	router := deliveryhttp.NewRouter(cfg, authUC, rateLimiter)
	httpServer := &http.Server{
		Addr:         fmt.Sprintf(":%s", cfg.App.Port),
		Handler:      router,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// ==========================================================
	// gRPC Server
	// ==========================================================
	grpcSrv := deliverygrpc.NewServer(authUC)
	grpcServer := deliverygrpc.NewGRPCServer(grpcSrv)

	// ==========================================================
	// Start servers (goroutine)
	// ==========================================================
	go func() {
		log.Info().Str("addr", httpServer.Addr).Msg("HTTP server started")
		if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatal().Err(err).Msg("HTTP server error")
		}
	}()

	go func() {
		log.Info().Str("port", cfg.App.GRPCPort).Msg("gRPC server started")
		if err := deliverygrpc.Listen(grpcServer, cfg.App.GRPCPort); err != nil {
			log.Fatal().Err(err).Msg("gRPC server error")
		}
	}()

	// ==========================================================
	// Graceful Shutdown
	// ==========================================================
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Info().Msg("shutting down servers...")

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer shutdownCancel()

	if err := httpServer.Shutdown(shutdownCtx); err != nil {
		log.Error().Err(err).Msg("HTTP server forced shutdown")
	}

	grpcServer.GracefulStop()

	log.Info().Msg("auth-service stopped")
}
