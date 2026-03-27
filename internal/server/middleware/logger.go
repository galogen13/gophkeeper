package middleware

import (
	"context"
	"time"

	"github.com/galogen13/gophkeeper/internal/logger"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/status"
)

func LoggingInterceptor() grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (any, error) {
		start := time.Now()

		// Обрабатываем запрос
		resp, err := handler(ctx, req)

		// Логируем результат
		duration := time.Since(start)
		st, _ := status.FromError(err)

		logger.Log.Info("gRPC request",
			zap.String("method", info.FullMethod),
			zap.String("duration", duration.String()),
			zap.String("code", st.Code().String()),
			zap.Error(err),
		)

		return resp, err
	}
}
