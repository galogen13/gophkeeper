package main

import (
	"flag"
	"fmt"
	"log"
	"net"
	"os"
	"os/signal"
	"syscall"

	"github.com/galogen13/gophkeeper/internal/buildinfo"
	"github.com/galogen13/gophkeeper/internal/logger"
	"github.com/galogen13/gophkeeper/internal/pkg/proto"
	"github.com/galogen13/gophkeeper/internal/server/config"
	"github.com/galogen13/gophkeeper/internal/server/crypto"
	"github.com/galogen13/gophkeeper/internal/server/handlers"
	"github.com/galogen13/gophkeeper/internal/server/middleware"
	"github.com/galogen13/gophkeeper/internal/server/repository/postgres"
	"github.com/galogen13/gophkeeper/internal/server/service"
	"go.uber.org/zap"
	"google.golang.org/grpc"
)

var (
	buildVersion = buildinfo.BuildInfoNotAvaluable
	buildDate    = buildinfo.BuildInfoNotAvaluable
)

func main() {

	buildinfo.PrintBuildInfo(buildVersion, buildDate)

	if err := run(); err != nil {
		log.Fatal(err)
	}
}

func run() error {
	configPath := flag.String("config", "configs/server.yaml", "path to config file")
	flag.Parse()

	cfg, err := config.Load(*configPath)
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	if err := logger.Initialize(cfg.Log.Level); err != nil {
		return fmt.Errorf("failed to init logger: %w", err)
	}
	defer logger.Log.Sync()

	// Подключение к БД
	db, err := postgres.ConnectDB(cfg.Database)
	if err != nil {
		return fmt.Errorf("failed to connect to database: %w", err)
	}
	defer db.Close()

	// Инициализация репозиториев
	userRepo := postgres.NewUserRepository(db)
	secretRepo := postgres.NewSecretRepository(db)
	txManager := postgres.NewTransactionManager(db)

	// Инициализация криптографии
	passwordHasher := crypto.NewPasswordHasher(crypto.DefaultPasswordConfig)
	jwtManager := crypto.NewJWTManager(
		cfg.JWT.Secret,
		cfg.JWT.AccessTokenTTL,
		cfg.JWT.RefreshTokenTTL,
	)

	// Инициализация сервисов
	authService := service.NewAuthService(userRepo, txManager, passwordHasher, jwtManager)
	secretService := service.NewSecretService(secretRepo, txManager)

	// Создаём middleware
	authMiddleware := middleware.NewAuthMiddleware(jwtManager)

	// Настраиваем gRPC сервер с перехватчиками
	grpcServer := grpc.NewServer(
		grpc.ChainUnaryInterceptor(
			middleware.LoggingInterceptor(),
			authMiddleware.AuthInterceptor(),
		),
	)

	// Регистрируем обработчики
	proto.RegisterAuthServiceServer(grpcServer, handlers.NewAuthHandler(authService))
	proto.RegisterKeeperServiceServer(grpcServer, handlers.NewKeeperHandler(secretService))

	// Включаем reflection для инструментов типа grpcurl (только для разработки)
	// reflection.Register(grpcServer)

	// Запускаем сервер
	listener, err := net.Listen("tcp", cfg.Server.Address)
	if err != nil {
		return fmt.Errorf("ailed to listen: %w", err)
	}

	logger.Log.Info("Server listening", zap.String("address", cfg.Server.Address))

	// Graceful shutdown
	go func() {
		if err := grpcServer.Serve(listener); err != nil {
			log.Fatalf("Failed to serve: %v", err)
		}
	}()

	// Ждём сигнала завершения
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logger.Log.Info("Shutting down server...")
	grpcServer.GracefulStop()
	logger.Log.Info("Server stopped")

	return nil
}
