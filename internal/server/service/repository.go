package service

import (
	"context"
	"time"

	"github.com/galogen13/gophkeeper/internal/pkg/models"

	"github.com/google/uuid"
)

type UserRepository interface {
	// Create сохраняет нового пользователя
	Create(ctx context.Context, user *models.User) error

	// GetByID получает пользователя по ID
	GetByID(ctx context.Context, id uuid.UUID) (*models.User, error)

	// GetByEmail получает пользователя по email
	GetByEmail(ctx context.Context, email string) (*models.User, error)

	// Update обновляет данные пользователя
	Update(ctx context.Context, user *models.User) error

	// Delete помечает пользователя как удалённого (soft delete)
	Delete(ctx context.Context, id uuid.UUID) error
}

type SecretRepository interface {
	// Create сохраняет новый секрет
	Create(ctx context.Context, secret *models.Secret) error

	// GetByID получает секрет по ID (только если владелец совпадает)
	GetByID(ctx context.Context, id, ownerID uuid.UUID) (*models.Secret, error)

	// ListByOwner возвращает все секреты владельца с пагинацией
	ListByOwner(ctx context.Context, ownerID uuid.UUID, limit, offset int, includeDeleted bool) ([]*models.Secret, int64, error)

	// Update обновляет секрет (с проверкой версии)
	Update(ctx context.Context, secret *models.Secret) error

	// Delete мягко удаляет секрет
	Delete(ctx context.Context, id, ownerID uuid.UUID) error

	// HardDelete полностью удаляет секрет (только для тестов/админа)
	HardDelete(ctx context.Context, id, ownerID uuid.UUID) error

	// GetChangedSince возвращает секреты, изменённые после указанного времени (для синхронизации)
	GetChangedSince(ctx context.Context, ownerID uuid.UUID, sinceTime *time.Time) ([]*models.Secret, error)
}

// Транзакции
type TransactionManager interface {
	// WithinTransaction выполняет функцию внутри транзакции
	WithinTransaction(ctx context.Context, fn func(ctx context.Context) error) error
}
