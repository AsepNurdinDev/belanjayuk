package grpc

import (
	"context"
	"errors"
	"net"
	"time"

	"github.com/rs/zerolog/log"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/keepalive"
	"google.golang.org/grpc/status"

	"github.com/AsepNurdinDev/belanjayuk/services/auth-service/internal/domain"
	"github.com/AsepNurdinDev/belanjayuk/services/auth-service/internal/usecase"
	authpb "github.com/AsepNurdinDev/belanjayuk/services/auth-service/proto/v1"
)

// Server — gRPC server untuk auth-service
// Digunakan oleh service lain untuk memvalidasi JWT access token
type Server struct {
	authpb.UnimplementedAuthServiceServer
	uc usecase.AuthUsecase
}

func NewServer(uc usecase.AuthUsecase) *Server {
	return &Server{uc: uc}
}

// NewGRPCServer — membuat gRPC server dengan konfigurasi keamanan
func NewGRPCServer(srv *Server) *grpc.Server {
	kaParams := keepalive.ServerParameters{
		MaxConnectionIdle:     5 * time.Minute,
		MaxConnectionAge:      10 * time.Minute,
		MaxConnectionAgeGrace: 5 * time.Second,
		Time:                  2 * time.Minute,
		Timeout:               20 * time.Second,
	}

	kaEnforcementPolicy := keepalive.EnforcementPolicy{
		MinTime:             30 * time.Second,
		PermitWithoutStream: true,
	}

	grpcServer := grpc.NewServer(
		grpc.KeepaliveParams(kaParams),
		grpc.KeepaliveEnforcementPolicy(kaEnforcementPolicy),
		grpc.UnaryInterceptor(loggingInterceptor),
		grpc.MaxRecvMsgSize(1*1024*1024),
		grpc.MaxSendMsgSize(1*1024*1024),
	)

	authpb.RegisterAuthServiceServer(grpcServer, srv)
	return grpcServer
}

// Listen — start gRPC server di port yang ditentukan
func Listen(grpcServer *grpc.Server, port string) error {
	lis, err := net.Listen("tcp", ":"+port)
	if err != nil {
		return err
	}
	return grpcServer.Serve(lis)
}

// =============================================================
// gRPC Handlers
// =============================================================

// ValidateToken — dipanggil service lain untuk validasi JWT
func (s *Server) ValidateToken(ctx context.Context, req *authpb.ValidateTokenRequest) (*authpb.ValidateTokenResponse, error) {
	if req.AccessToken == "" {
		return nil, status.Error(codes.InvalidArgument, "access_token is required")
	}

	claims, err := s.uc.ValidateToken(ctx, req.AccessToken)
	if err != nil {
		return nil, mapDomainError(err)
	}

	return &authpb.ValidateTokenResponse{
		UserId: claims.UserID,
		Email:  claims.Email,
		Role:   string(claims.Role),
	}, nil
}

// =============================================================
// Error Mapping
// =============================================================

func mapDomainError(err error) error {
	switch {
	case errors.Is(err, domain.ErrInvalidToken):
		return status.Error(codes.Unauthenticated, "invalid token")
	case errors.Is(err, domain.ErrExpiredToken):
		return status.Error(codes.Unauthenticated, "token expired")
	case errors.Is(err, domain.ErrRevokedToken):
		return status.Error(codes.Unauthenticated, "token revoked")
	case errors.Is(err, domain.ErrTokenBlacklist):
		return status.Error(codes.Unauthenticated, "token invalidated")
	case errors.Is(err, domain.ErrUnauthorized):
		return status.Error(codes.PermissionDenied, "unauthorized")
	default:
		return status.Error(codes.Internal, "internal error")
	}
}

// =============================================================
// Interceptors
// =============================================================

// loggingInterceptor — structured log setiap gRPC call dengan zerolog
func loggingInterceptor(
	ctx context.Context,
	req interface{},
	info *grpc.UnaryServerInfo,
	handler grpc.UnaryHandler,
) (interface{}, error) {
	start := time.Now()
	resp, err := handler(ctx, req)
	duration := time.Since(start)

	event := log.Info()
	if err != nil {
		event = log.Error().Err(err)
	}
	event.
		Str("method", info.FullMethod).
		Dur("duration_ms", duration).
		Msg("grpc call")

	return resp, err
}
