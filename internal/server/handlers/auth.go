package handlers

import (
	"context"

	"github.com/galogen13/gophkeeper/internal/logger"
	"go.uber.org/zap"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/galogen13/gophkeeper/internal/pkg/proto"
	"github.com/galogen13/gophkeeper/internal/server/service"
)

type AuthHandler struct {
	proto.UnimplementedAuthServiceServer
	authService *service.AuthService
}

func NewAuthHandler(authService *service.AuthService) *AuthHandler {
	return &AuthHandler{
		authService: authService,
	}
}

// Register обрабатывает регистрацию нового пользователя
func (h *AuthHandler) Register(ctx context.Context, req *proto.RegisterRequest) (*proto.AuthResponse, error) {
	if req.GetEmail() == "" || req.GetPassword() == "" {
		return nil, status.Error(codes.InvalidArgument, "email and password are required")
	}

	output, err := h.authService.Register(ctx, service.RegisterInput{
		Email:    req.GetEmail(),
		Password: req.GetPassword(),
	})
	if err != nil {
		logger.Log.Error("failed to register user", zap.Error(err), zap.String("email", req.GetEmail()))
		return nil, mapErrorToGRPC(err)
	}

	logger.Log.Info("user registered successfully", zap.String("user_id", output.UserID.String()))

	pb := proto.AuthResponse_builder{}.Build()
	pb.SetAccessToken(output.AccessToken)
	pb.SetRefreshToken(output.RefreshToken)
	pb.SetExpiresIn(int64(output.ExpiresIn.Seconds()))

	return pb, nil
}

// Login обрабатывает вход пользователя
func (h *AuthHandler) Login(ctx context.Context, req *proto.LoginRequest) (*proto.AuthResponse, error) {
	if req.GetEmail() == "" || req.GetPassword() == "" {
		return nil, status.Error(codes.InvalidArgument, "email and password are required")
	}

	output, err := h.authService.Login(ctx, service.LoginInput{
		Email:    req.GetEmail(),
		Password: req.GetPassword(),
	})
	if err != nil {
		logger.Log.Error("failed to login", zap.Error(err), zap.String("email", req.GetEmail()))
		return nil, mapErrorToGRPC(err)
	}

	logger.Log.Info("user logged in", zap.String("user_id", output.UserID.String()))

	pb := proto.AuthResponse_builder{}.Build()
	pb.SetAccessToken(output.AccessToken)
	pb.SetRefreshToken(output.RefreshToken)
	pb.SetExpiresIn(int64(output.ExpiresIn.Seconds()))

	return pb, nil
}

// RefreshToken обновляет пару токенов
func (h *AuthHandler) RefreshToken(ctx context.Context, req *proto.RefreshTokenRequest) (*proto.AuthResponse, error) {
	if req.GetRefreshToken() == "" {
		return nil, status.Error(codes.InvalidArgument, "refresh token is required")
	}

	output, err := h.authService.RefreshToken(ctx, req.GetRefreshToken())
	if err != nil {
		logger.Log.Error("failed to refresh token", zap.Error(err))
		return nil, mapErrorToGRPC(err)
	}

	pb := proto.AuthResponse_builder{}.Build()
	pb.SetAccessToken(output.AccessToken)
	pb.SetRefreshToken(output.RefreshToken)
	pb.SetExpiresIn(int64(output.ExpiresIn.Seconds()))

	return pb, nil
}
