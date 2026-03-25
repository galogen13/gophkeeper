package middleware

import (
	"context"
	"strings"

	"github.com/google/uuid"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"

	"github.com/galogen13/gophkeeper/internal/server/crypto"
)

type AuthMiddleware struct {
	jwtManager *crypto.JWTManager
}

func NewAuthMiddleware(jwtManager *crypto.JWTManager) *AuthMiddleware {
	return &AuthMiddleware{jwtManager: jwtManager}
}

// UnaryInterceptor проверяет JWT токен для унитарных запросов
func (m *AuthMiddleware) AuthInterceptor() grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
		// Пропускаем методы аутентификации (они не требуют токена)
		if m.isPublicMethod(info.FullMethod) {
			return handler(ctx, req)
		}

		// Извлекаем токен из метаданных
		token, err := m.extractToken(ctx)
		if err != nil {
			return nil, err
		}

		// Проверяем токен
		claims, err := m.jwtManager.VerifyToken(token)
		if err != nil {
			return nil, status.Error(codes.Unauthenticated, "invalid token")
		}

		// Добавляем user_id в контекст для дальнейшего использования
		ctx = context.WithValue(ctx, userIDKey{}, claims.UserID)

		return handler(ctx, req)
	}
}

// isPublicMethod проверяет, требует ли метод аутентификации
func (m *AuthMiddleware) isPublicMethod(method string) bool {
	publicMethods := []string{
		"/gophkeeper.AuthService/Register",
		"/gophkeeper.AuthService/Login",
		"/gophkeeper.AuthService/RefreshToken",
	}

	for _, m := range publicMethods {
		if m == method {
			return true
		}
	}
	return false
}

// extractToken извлекает JWT токен из метаданных
func (m *AuthMiddleware) extractToken(ctx context.Context) (string, error) {
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return "", status.Error(codes.Unauthenticated, "metadata not provided")
	}

	values := md.Get("authorization")
	if len(values) == 0 {
		return "", status.Error(codes.Unauthenticated, "authorization token not provided")
	}

	token := values[0]
	// Удаляем префикс "Bearer " если есть
	token = strings.TrimPrefix(token, "Bearer ")

	if token == "" {
		return "", status.Error(codes.Unauthenticated, "empty token")
	}

	return token, nil
}

type userIDKey struct{}

// GetUserID извлекает user_id из контекста
func GetUserID(ctx context.Context) (uuid.UUID, error) {
	val := ctx.Value(userIDKey{})
	if val == nil {
		return uuid.Nil, status.Error(codes.Unauthenticated, "user not authenticated")
	}

	userID, ok := val.(uuid.UUID)
	if !ok {
		return uuid.Nil, status.Error(codes.Internal, "invalid user id in context")
	}

	return userID, nil
}
