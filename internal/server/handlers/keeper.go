package handlers

import (
	"context"
	"time"

	"github.com/galogen13/gophkeeper/internal/logger"
	"github.com/galogen13/gophkeeper/internal/server"
	"github.com/google/uuid"
	"go.uber.org/zap"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"

	"github.com/galogen13/gophkeeper/internal/pkg/proto"
	"github.com/galogen13/gophkeeper/internal/server/middleware"
	"github.com/galogen13/gophkeeper/internal/server/service"
)

type KeeperHandler struct {
	proto.UnimplementedKeeperServiceServer
	secretService *service.SecretService
}

func NewKeeperHandler(secretService *service.SecretService) *KeeperHandler {
	return &KeeperHandler{
		secretService: secretService,
	}
}

// CreateSecret создаёт новый секрет
func (h *KeeperHandler) CreateSecret(ctx context.Context, req *proto.CreateSecretRequest) (*proto.Secret, error) {
	// Получаем ID пользователя из контекста (установлен middleware)
	userID, err := middleware.GetUserID(ctx)
	if err != nil {
		return nil, err
	}

	secretID := uuid.Nil
	if req.GetId() != "" {
		secretID, err = uuid.Parse(req.GetId())
		if err != nil {
			return nil, status.Error(codes.InvalidArgument, "invalid secret id format")
		}
	}

	if req.GetTitle() == "" {
		return nil, status.Error(codes.InvalidArgument, "title is required")
	}

	if len(req.GetEncryptedData()) == 0 {
		return nil, status.Error(codes.InvalidArgument, "encrypted data is required")
	}

	secret, err := h.secretService.Create(ctx, service.CreateSecretInput{
		OwnerID:       userID,
		Id:            secretID,
		Type:          server.SecretType(req.GetType()),
		Title:         req.GetTitle(),
		EncryptedData: req.GetEncryptedData(),
		Meta:          req.GetMeta(),
	})
	if err != nil {
		logger.Log.Error("failed to create secret", zap.Error(err), zap.String("user_id", userID.String()))
		return nil, mapErrorToGRPC(err)
	}
	logger.Log.Info("failed to create secret", zap.String("user_id", userID.String()), zap.String("secret_id", secret.ID.String()))
	return secretToProtoSecret(secret), nil
}

// GetSecret получает секрет по ID
func (h *KeeperHandler) GetSecret(ctx context.Context, req *proto.GetSecretRequest) (*proto.Secret, error) {
	userID, err := middleware.GetUserID(ctx)
	if err != nil {
		return nil, err
	}

	if req.GetId() == "" {
		return nil, status.Error(codes.InvalidArgument, "secret id is required")
	}

	secretID, err := uuid.Parse(req.GetId())
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid secret id format")
	}

	secret, err := h.secretService.GetByID(ctx, secretID, userID)
	if err != nil {
		logger.Log.Error("failed to get secret", zap.Error(err), zap.String("secret_id", req.GetId()))

		return nil, mapErrorToGRPC(err)
	}

	return secretToProtoSecret(secret), nil
}

// ListSecrets возвращает список секретов пользователя
func (h *KeeperHandler) ListSecrets(ctx context.Context, req *proto.ListSecretsRequest) (*proto.ListSecretsResponse, error) {
	userID, err := middleware.GetUserID(ctx)
	if err != nil {
		return nil, err
	}

	// Пагинация
	page := int(req.GetPage())
	if page < 1 {
		page = 1
	}
	pageSize := int(req.GetPageSize())
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}
	offset := (page - 1) * pageSize

	secrets, total, err := h.secretService.ListByOwner(ctx, userID, pageSize, offset, req.GetIncludeDeleted())
	if err != nil {
		logger.Log.Error("failed to list secrets", zap.Error(err), zap.String("user_id", userID.String()))
		return nil, mapErrorToGRPC(err)
	}

	pbSecrets := make([]*proto.Secret, len(secrets))
	for i, s := range secrets {
		pbSecrets[i] = secretToProtoSecret(s)
	}

	pb := proto.ListSecretsResponse_builder{}.Build()
	pb.SetSecrets(pbSecrets)
	pb.SetTotal(int32(total))
	pb.SetPage(int32(page))

	return pb, nil
}

// UpdateSecret обновляет существующий секрет
func (h *KeeperHandler) UpdateSecret(ctx context.Context, req *proto.UpdateSecretRequest) (*proto.Secret, error) {
	userID, err := middleware.GetUserID(ctx)
	if err != nil {
		return nil, err
	}

	if req.GetId() == "" {
		return nil, status.Error(codes.InvalidArgument, "secret id is required")
	}

	secretID, err := uuid.Parse(req.GetId())
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid secret id format")
	}

	secret, err := h.secretService.Update(ctx, service.UpdateSecretInput{
		ID:            secretID,
		OwnerID:       userID,
		Title:         req.GetTitle(),
		EncryptedData: req.GetEncryptedData(),
		Meta:          req.GetMeta(),
	})
	if err != nil {
		logger.Log.Error("failed to update secret", zap.Error(err), zap.String("secret_id", req.GetId()))

		return nil, mapErrorToGRPC(err)
	}

	return secretToProtoSecret(secret), nil
}

// DeleteSecret удаляет секрет
func (h *KeeperHandler) DeleteSecret(ctx context.Context, req *proto.DeleteSecretRequest) (*emptypb.Empty, error) {
	userID, err := middleware.GetUserID(ctx)
	if err != nil {
		return nil, err
	}

	if req.GetId() == "" {
		return nil, status.Error(codes.InvalidArgument, "secret id is required")
	}

	secretID, err := uuid.Parse(req.GetId())
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid secret id format")
	}

	err = h.secretService.Delete(ctx, secretID, userID, req.GetPermanent())
	if err != nil {
		logger.Log.Error("failed to delete secret", zap.Error(err), zap.String("secret_id", req.GetId()))

		return nil, mapErrorToGRPC(err)
	}

	return &emptypb.Empty{}, nil
}

// SyncSecrets стриминг для синхронизации данных
func (h *KeeperHandler) SyncSecrets(ctx context.Context, req *proto.SyncRequest) (*proto.SyncResponse, error) {
	userID, err := middleware.GetUserID(ctx)
	if err != nil {
		return nil, err
	}

	var lastSync *time.Time
	if req.GetLastSync() != nil {
		t := req.GetLastSync().AsTime()
		lastSync = &t
	}

	// Получаем изменения
	changes, err := h.secretService.GetChanges(ctx, userID, lastSync)
	if err != nil {
		return nil, err
	}

	// Разделяем на активные и удалённые
	secrets := make([]*proto.Secret, 0, len(changes))
	deletedIDs := make([]string, 0, len(changes))

	for _, s := range changes {
		if s.DeletedAt != nil {
			deletedIDs = append(deletedIDs, s.ID.String())
		} else {
			secrets = append(secrets, secretToProtoSecret(s))
		}
	}

	logger.Log.Info("sync completed", zap.String("user_id", userID.String()), zap.Int("secrets", len(secrets)), zap.Int("deleted", len(deletedIDs)))

	pb := proto.SyncResponse_builder{}.Build()
	pb.SetSecrets(secrets)
	pb.SetDeletedIds(deletedIDs)

	return pb, nil
}
