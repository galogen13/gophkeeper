package grpc

import (
	"time"

	"github.com/galogen13/gophkeeper/internal/client"
	"github.com/galogen13/gophkeeper/internal/pkg/proto"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// Конвертация из proto в клиентскую модель
func SecretFromProtoSecret(pb *proto.Secret) *client.Secret {
	if pb == nil {
		return nil
	}

	secret := &client.Secret{
		ID:            pb.GetId(),
		Type:          client.SecretType(pb.GetType()),
		Title:         pb.GetTitle(),
		EncryptedData: pb.GetEncryptedData(),
		Meta:          pb.GetMeta(),
		Version:       pb.GetVersion(),
		CreatedAt:     pb.GetCreatedAt().AsTime(),
		IsDeleted:     pb.GetIsDeleted(),
	}

	if pb.GetUpdatedAt() != nil {
		t := pb.GetUpdatedAt().AsTime()
		secret.UpdatedAt = &t
	}

	return secret
}

// Конвертация из клиентской модели в proto для отправки на сервер
func SecretToProtoSecretCreate(s *client.Secret) *proto.CreateSecretRequest {
	pb := proto.CreateSecretRequest_builder{}.Build()
	pb.SetType(proto.SecretType(s.Type))
	pb.SetTitle(s.Title)
	pb.SetEncryptedData(s.EncryptedData)
	pb.SetMeta(s.Meta)
	return pb
}

func SecretToProtoSecretUpdate(s *client.Secret) *proto.UpdateSecretRequest {
	pb := proto.UpdateSecretRequest_builder{}.Build()
	pb.SetId(s.ID)
	pb.SetTitle(s.Title)
	pb.SetEncryptedData(s.EncryptedData)
	pb.SetMeta(s.Meta)
	pb.SetVersion(s.Version)
	return pb
}

func DeleteSecretToProtoDeleteSecret(d *client.DeleteSecret) *proto.DeleteSecretRequest {
	pb := proto.DeleteSecretRequest_builder{}.Build()
	pb.SetId(d.ID)
	pb.SetPermanent(d.Permanent)

	return pb
}

// Конвертация AuthInfo
func AuthFromProtoAuth(pb *proto.AuthResponse, userID string) *client.AuthInfo {
	return &client.AuthInfo{
		UserID:       userID,
		AccessToken:  pb.GetAccessToken(),
		RefreshToken: pb.GetRefreshToken(),
		ExpiresIn:    pb.GetExpiresIn(),
		SavedAt:      time.Now(),
	}
}

func RegisterInfoToProtoRegisterRequest(r *client.RegisterLoginInfo) *proto.RegisterRequest {
	pb := proto.RegisterRequest_builder{}.Build()
	pb.SetEmail(r.Email)
	pb.SetPassword(r.Password)
	return pb
}

func RegisterInfoToProtoLoginRequest(r *client.RegisterLoginInfo) *proto.LoginRequest {
	pb := proto.LoginRequest_builder{}.Build()
	pb.SetEmail(r.Email)
	pb.SetPassword(r.Password)
	return pb
}

func SyncInfoToProtoSyncRequest(s *client.SyncInfo) *proto.SyncRequest {
	pb := proto.SyncRequest_builder{}.Build()
	if s.LastSync != nil {
		pb.SetLastSync(timestamppb.New(*s.LastSync))
	}

	return pb
}
