package service

import (
	"context"
	"errors"
	"time"

	"github.com/galogen13/gophkeeper/internal/server"
	"github.com/galogen13/gophkeeper/internal/server/repository"
	"github.com/google/uuid"
)

type SecretService struct {
	secretRepo SecretRepository
	txManager  TransactionManager
}

func NewSecretService(
	secretRepo SecretRepository,
	txManager TransactionManager,
) *SecretService {
	return &SecretService{
		secretRepo: secretRepo,
		txManager:  txManager,
	}
}

type CreateSecretInput struct {
	Id            uuid.UUID
	OwnerID       uuid.UUID
	Type          server.SecretType
	Title         string
	EncryptedData []byte
	Meta          string
}

func (s *SecretService) Create(ctx context.Context, input CreateSecretInput) (*server.Secret, error) {
	if input.Title == "" {
		return nil, errors.New("title is required")
	}

	if len(input.EncryptedData) == 0 {
		return nil, errors.New("encrypted data is required")
	}

	secret := &server.Secret{
		ID:            input.Id,
		OwnerID:       input.OwnerID,
		Type:          input.Type,
		Title:         input.Title,
		EncryptedData: input.EncryptedData,
		Meta:          input.Meta,
		Version:       1,
		CreatedAt:     time.Now(),
	}

	err := s.secretRepo.Create(ctx, secret)
	if err != nil {
		return nil, err
	}

	return secret, nil
}

func (s *SecretService) GetByID(ctx context.Context, id, ownerID uuid.UUID) (*server.Secret, error) {
	return s.secretRepo.GetByID(ctx, id, ownerID)
}

func (s *SecretService) ListByOwner(ctx context.Context, ownerID uuid.UUID, limit, offset int, includeDeleted bool) ([]*server.Secret, int64, error) {
	return s.secretRepo.ListByOwner(ctx, ownerID, limit, offset, includeDeleted)
}

type UpdateSecretInput struct {
	ID            uuid.UUID
	OwnerID       uuid.UUID
	Title         string
	EncryptedData []byte
	Meta          string
	Version       int64
}

func (s *SecretService) Update(ctx context.Context, input UpdateSecretInput) (*server.Secret, error) {

	existing, err := s.secretRepo.GetByID(ctx, input.ID, input.OwnerID)
	if err != nil {
		return nil, err
	}

	if existing.DeletedAt != nil {
		return nil, repository.ErrSecretDeleted
	}

	secret := &server.Secret{
		ID:            input.ID,
		OwnerID:       input.OwnerID,
		Title:         input.Title,
		EncryptedData: input.EncryptedData,
		Meta:          input.Meta,
		Version:       input.Version,
	}

	err = s.secretRepo.Update(ctx, secret)
	if err != nil {
		return nil, err
	}

	return secret, nil
}

func (s *SecretService) Delete(ctx context.Context, id, ownerID uuid.UUID, permanent bool) error {
	if permanent {
		return s.secretRepo.HardDelete(ctx, id, ownerID)
	}
	return s.secretRepo.Delete(ctx, id, ownerID)
}

func (s *SecretService) GetChanges(ctx context.Context, ownerID uuid.UUID, lastSync *time.Time) ([]*server.Secret, error) {
	if lastSync == nil {
		// Если синхронизация первый раз, возвращаем всё
		secrets, _, err := s.secretRepo.ListByOwner(ctx, ownerID, 1000, 0, true)
		return secrets, err
	}

	return s.secretRepo.GetChangedSince(ctx, ownerID, lastSync)
}
