package client

import (
	"context"
	"time"
)

type Storage interface {
	// Auth operations
	SaveAuth(ctx context.Context, auth *AuthInfo) error
	GetAuth(ctx context.Context) (*AuthInfo, error)
	ClearAuth(ctx context.Context) error

	// Secret operations
	// SaveSecret(ctx context.Context, secret *Secret) error
	SaveSecrets(ctx context.Context, secrets []*Secret) error // для синхронизации
	GetSecret(ctx context.Context, id string) (*Secret, error)
	ListSecrets(ctx context.Context) ([]*Secret, error)
	ListActiveSecrets(ctx context.Context) ([]*Secret, error) // только не удалённые
	DeleteSecret(ctx context.Context, id string) error
	// UpdateSecret(ctx context.Context, secret *Secret) error

	// Sync operations
	GetLastSyncTime(ctx context.Context) (*time.Time, error)
	SetLastSyncTime(ctx context.Context, t time.Time) error

	// Close
	Close() error
}
