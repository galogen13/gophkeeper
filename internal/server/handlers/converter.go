package handlers

import (
	"errors"

	"github.com/galogen13/gophkeeper/internal/server/repository"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/galogen13/gophkeeper/internal/pkg/proto"
	"github.com/galogen13/gophkeeper/internal/server"
)

// Конвертация моделей в proto-сообщения
func secretToProtoSecret(secret *server.Secret) *proto.Secret {
	pb := proto.Secret_builder{}.Build()
	pb.SetId(secret.ID.String())
	pb.SetOwnerId(secret.OwnerID.String())
	pb.SetType(proto.SecretType(secret.Type))
	pb.SetTitle(secret.Title)
	pb.SetEncryptedData(secret.EncryptedData)
	pb.SetMeta(secret.Meta)
	pb.SetCreatedAt(timestamppb.New(secret.CreatedAt))
	pb.SetIsDeleted(secret.DeletedAt != nil)

	if secret.UpdatedAt != nil {
		pb.SetUpdatedAt(timestamppb.New(*secret.UpdatedAt))
	}

	return pb
}

// Конвертация ошибок в gRPC статусы
func mapErrorToGRPC(err error) error {
	switch {
	case errors.Is(err, repository.ErrUserNotFound):
		return status.Error(codes.NotFound, "user not found")
	case errors.Is(err, repository.ErrEmailAlreadyExists):
		return status.Error(codes.AlreadyExists, "email already exists")
	case errors.Is(err, repository.ErrInvalidCredentials):
		return status.Error(codes.Unauthenticated, "invalid credentials")
	case errors.Is(err, repository.ErrSecretNotFound):
		return status.Error(codes.NotFound, "secret not found")
	case errors.Is(err, repository.ErrSecretDeleted):
		return status.Error(codes.FailedPrecondition, "secret is deleted")
	case errors.Is(err, repository.ErrVersionMismatch):
		return status.Error(codes.FailedPrecondition, "version mismatch")
	default:
		return status.Error(codes.Internal, "internal server error")
	}
}
