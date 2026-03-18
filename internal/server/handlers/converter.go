package handlers

import (
	"time"

	"github.com/galogen13/gophkeeper/internal/server/repository"
	"github.com/google/uuid"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/galogen13/gophkeeper/internal/pkg/models"
	"github.com/galogen13/gophkeeper/internal/pkg/proto"
)

// Конвертация моделей в proto-сообщения
func secretToProtoSecret(secret *models.Secret) *proto.Secret {
	pb := proto.Secret_builder{}.Build()
	pb.SetId(secret.ID.String())
	pb.SetOwnerId(secret.OwnerID.String())
	pb.SetType(proto.SecretType(secret.Type))
	pb.SetTitle(secret.Title)
	pb.SetEncryptedData(secret.EncryptedData)
	pb.SetMeta(secret.Meta)
	pb.SetVersion(secret.Version)
	pb.SetCreatedAt(timestamppb.New(secret.CreatedAt))
	pb.SetIsDeleted(secret.DeletedAt != nil)

	if secret.UpdatedAt != nil {
		pb.SetUpdatedAt(timestamppb.New(*secret.UpdatedAt))
	}

	return pb
}

// Конвертация proto в модель для создания
func protoSecretToSecretCreate(req *proto.CreateSecretRequest, ownerID uuid.UUID) *models.Secret {

	return &models.Secret{
		ID:            uuid.New(),
		OwnerID:       ownerID,
		Type:          models.SecretType(req.GetType()),
		Title:         req.GetTitle(),
		EncryptedData: req.GetEncryptedData(),
		Meta:          req.GetMeta(),
		Version:       1,
		CreatedAt:     time.Now(),
	}

}

// Конвертация proto в модель для обновления
func protoSecretToSecretUpdate(req *proto.UpdateSecretRequest, ownerID uuid.UUID) *models.Secret {
	id, _ := uuid.Parse(req.GetId())
	return &models.Secret{
		ID:            id,
		OwnerID:       ownerID,
		Title:         req.GetTitle(),
		EncryptedData: req.GetEncryptedData(),
		Meta:          req.GetMeta(),
		Version:       req.GetVersion(),
	}
}

// Конвертация ошибок в gRPC статусы
func mapErrorToGRPC(err error) error {
	switch err {
	case repository.ErrUserNotFound:
		return status.Error(codes.NotFound, "user not found")
	case repository.ErrEmailAlreadyExists:
		return status.Error(codes.AlreadyExists, "email already exists")
	case repository.ErrInvalidCredentials:
		return status.Error(codes.Unauthenticated, "invalid credentials")
	case repository.ErrSecretNotFound:
		return status.Error(codes.NotFound, "secret not found")
	case repository.ErrSecretDeleted:
		return status.Error(codes.FailedPrecondition, "secret is deleted")
	case repository.ErrVersionMismatch:
		return status.Error(codes.FailedPrecondition, "version mismatch")
	default:
		return status.Error(codes.Internal, "internal server error")
	}
}
