package service

import (
	"context"
	"errors"
	"time"

	"github.com/galogen13/gophkeeper/internal/pkg/models"
	"github.com/galogen13/gophkeeper/internal/server/crypto"
	"github.com/galogen13/gophkeeper/internal/server/repository"

	"github.com/google/uuid"
)

type AuthService struct {
	userRepo   UserRepository
	txManager  TransactionManager
	hasher     *crypto.PasswordHasher
	jwtManager *crypto.JWTManager
}

func NewAuthService(
	userRepo UserRepository,
	txManager TransactionManager,
	hasher *crypto.PasswordHasher,
	jwtManager *crypto.JWTManager,
) *AuthService {
	return &AuthService{
		userRepo:   userRepo,
		txManager:  txManager,
		hasher:     hasher,
		jwtManager: jwtManager,
	}
}

type RegisterInput struct {
	Email    string
	Password string
}

type AuthOutput struct {
	UserID       uuid.UUID
	AccessToken  string
	RefreshToken string
	ExpiresIn    time.Duration
}

func (s *AuthService) Register(ctx context.Context, input RegisterInput) (*AuthOutput, error) {
	if input.Email == "" || input.Password == "" {
		return nil, errors.New("email and password are required")
	}

	// Хешируем пароль
	passwordHash, err := s.hasher.HashPassword(input.Password)
	if err != nil {
		return nil, err
	}

	var user *models.User
	var tokenPair *crypto.TokenPair

	// В транзакции создаём пользователя и генерируем токены
	err = s.txManager.WithinTransaction(ctx, func(txCtx context.Context) error {
		user = &models.User{
			ID:           uuid.New(),
			Email:        input.Email,
			PasswordHash: []byte(passwordHash),
			CreatedAt:    time.Now(),
		}

		if err := s.userRepo.Create(txCtx, user); err != nil {
			return err
		}

		// Генерируем токены
		tokenPair, err = s.jwtManager.GenerateTokenPair(user.ID)
		return err
	})

	if err != nil {
		return nil, err
	}

	return &AuthOutput{
		UserID:       user.ID,
		AccessToken:  tokenPair.AccessToken,
		RefreshToken: tokenPair.RefreshToken,
		ExpiresIn:    tokenPair.ExpiresIn,
	}, nil
}

type LoginInput struct {
	Email    string
	Password string
}

func (s *AuthService) Login(ctx context.Context, input LoginInput) (*AuthOutput, error) {
	// Получаем пользователя по email
	user, err := s.userRepo.GetByEmail(ctx, input.Email)
	if err != nil {
		if errors.Is(err, repository.ErrUserNotFound) {
			return nil, repository.ErrInvalidCredentials
		}
		return nil, err
	}

	// Проверяем пароль
	valid, err := s.hasher.VerifyPassword(input.Password, string(user.PasswordHash))
	if err != nil {
		return nil, err
	}
	if !valid {
		return nil, repository.ErrInvalidCredentials
	}

	// Генерируем токены
	tokenPair, err := s.jwtManager.GenerateTokenPair(user.ID)
	if err != nil {
		return nil, err
	}

	return &AuthOutput{
		UserID:       user.ID,
		AccessToken:  tokenPair.AccessToken,
		RefreshToken: tokenPair.RefreshToken,
		ExpiresIn:    tokenPair.ExpiresIn,
	}, nil
}

func (s *AuthService) RefreshToken(ctx context.Context, refreshToken string) (*AuthOutput, error) {
	// Проверяем refresh token
	claims, err := s.jwtManager.VerifyToken(refreshToken)
	if err != nil {
		return nil, err
	}

	// Генерируем новую пару токенов
	tokenPair, err := s.jwtManager.GenerateTokenPair(claims.UserID)
	if err != nil {
		return nil, err
	}

	return &AuthOutput{
		UserID:       claims.UserID,
		AccessToken:  tokenPair.AccessToken,
		RefreshToken: tokenPair.RefreshToken,
		ExpiresIn:    tokenPair.ExpiresIn,
	}, nil
}
