package postgres

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/galogen13/gophkeeper/internal/pkg/models"
	"github.com/galogen13/gophkeeper/internal/server/repository"

	"github.com/google/uuid"
)

type secretRepository struct {
	db *sql.DB
}

func NewSecretRepository(db *sql.DB) *secretRepository {
	return &secretRepository{db: db}
}

func (r *secretRepository) Create(ctx context.Context, secret *models.Secret) error {
	query := `
        INSERT INTO secrets (id, owner_id, type, title, encrypted_data, meta, version, created_at)
        VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
    `

	if secret.ID == uuid.Nil {
		secret.ID = uuid.New()
	}
	if secret.Version == 0 {
		secret.Version = 1
	}

	_, err := r.db.ExecContext(ctx, query,
		secret.ID, secret.OwnerID, secret.Type, secret.Title,
		secret.EncryptedData, secret.Meta, secret.Version, secret.CreatedAt,
	)

	return err
}

func (r *secretRepository) GetByID(ctx context.Context, id, ownerID uuid.UUID) (*models.Secret, error) {
	query := `
        SELECT id, owner_id, type, title, encrypted_data, meta, version, created_at, updated_at, deleted_at
        FROM secrets
        WHERE id = $1 AND owner_id = $2
    `

	secret := &models.Secret{}
	var updatedAt, deletedAt sql.NullTime

	err := r.db.QueryRowContext(ctx, query, id, ownerID).Scan(
		&secret.ID, &secret.OwnerID, &secret.Type, &secret.Title,
		&secret.EncryptedData, &secret.Meta, &secret.Version,
		&secret.CreatedAt, &updatedAt, &deletedAt,
	)

	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, repository.ErrSecretNotFound
		}
		return nil, err
	}

	if updatedAt.Valid {
		secret.UpdatedAt = &updatedAt.Time
	}
	if deletedAt.Valid {
		secret.DeletedAt = &deletedAt.Time
	}

	return secret, nil
}

func (r *secretRepository) ListByOwner(ctx context.Context, ownerID uuid.UUID, limit, offset int, includeDeleted bool) ([]*models.Secret, int64, error) {
	// Сначала получаем общее количество
	countQuery := `SELECT COUNT(*) FROM secrets WHERE owner_id = $1`
	if !includeDeleted {
		countQuery += ` AND deleted_at IS NULL`
	}

	var total int64
	err := r.db.QueryRowContext(ctx, countQuery, ownerID).Scan(&total)
	if err != nil {
		return nil, 0, err
	}

	// Затем получаем данные с пагинацией
	query := `
        SELECT id, owner_id, type, title, encrypted_data, meta, version, created_at, updated_at, deleted_at
        FROM secrets
        WHERE owner_id = $1
    `
	if !includeDeleted {
		query += ` AND deleted_at IS NULL`
	}
	query += ` ORDER BY created_at DESC LIMIT $2 OFFSET $3`

	rows, err := r.db.QueryContext(ctx, query, ownerID, limit, offset)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var secrets []*models.Secret
	for rows.Next() {
		secret := &models.Secret{}
		var updatedAt, deletedAt sql.NullTime

		err := rows.Scan(
			&secret.ID, &secret.OwnerID, &secret.Type, &secret.Title,
			&secret.EncryptedData, &secret.Meta, &secret.Version,
			&secret.CreatedAt, &updatedAt, &deletedAt,
		)
		if err != nil {
			return nil, 0, err
		}

		if updatedAt.Valid {
			secret.UpdatedAt = &updatedAt.Time
		}
		if deletedAt.Valid {
			secret.DeletedAt = &deletedAt.Time
		}

		secrets = append(secrets, secret)
	}

	return secrets, total, rows.Err()
}

func (r *secretRepository) Update(ctx context.Context, secret *models.Secret) error {
	query := `
        UPDATE secrets
        SET title = $1, encrypted_data = $2, meta = $3, version = version + 1, updated_at = $4
        WHERE id = $5 AND owner_id = $6 AND version = $7 AND deleted_at IS NULL
    `

	now := time.Now()
	result, err := r.db.ExecContext(ctx, query,
		secret.Title, secret.EncryptedData, secret.Meta, now,
		secret.ID, secret.OwnerID, secret.Version,
	)

	if err != nil {
		return err
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return err
	}

	if rows == 0 {
		// Проверяем, существует ли секрет и не удалён ли он
		existing, err := r.GetByID(ctx, secret.ID, secret.OwnerID)
		if err != nil {
			return err
		}
		if existing.DeletedAt != nil {
			return repository.ErrSecretDeleted
		}
		return repository.ErrVersionMismatch
	}

	secret.Version++
	secret.UpdatedAt = &now
	return nil
}

func (r *secretRepository) Delete(ctx context.Context, id, ownerID uuid.UUID) error {
	query := `
        UPDATE secrets
        SET deleted_at = $1
        WHERE id = $2 AND owner_id = $3 AND deleted_at IS NULL
    `

	result, err := r.db.ExecContext(ctx, query, time.Now(), id, ownerID)
	if err != nil {
		return err
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return err
	}

	if rows == 0 {
		return repository.ErrSecretNotFound
	}

	return nil
}

func (r *secretRepository) HardDelete(ctx context.Context, id, ownerID uuid.UUID) error {
	query := `DELETE FROM secrets WHERE id = $1 AND owner_id = $2`

	result, err := r.db.ExecContext(ctx, query, id, ownerID)
	if err != nil {
		return err
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return err
	}

	if rows == 0 {
		return repository.ErrSecretNotFound
	}

	return nil
}

func (r *secretRepository) GetChangedSince(ctx context.Context, ownerID uuid.UUID, sinceTime *time.Time) ([]*models.Secret, error) {
	query := `
        SELECT id, owner_id, type, title, encrypted_data, meta, version, created_at, updated_at, deleted_at
        FROM secrets
        WHERE owner_id = $1 AND (created_at > $2 OR updated_at > $2 OR deleted_at > $2)
        ORDER BY updated_at ASC
    `

	rows, err := r.db.QueryContext(ctx, query, ownerID, sinceTime)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var secrets []*models.Secret
	for rows.Next() {
		secret := &models.Secret{}
		var updatedAt, deletedAt sql.NullTime

		err := rows.Scan(
			&secret.ID, &secret.OwnerID, &secret.Type, &secret.Title,
			&secret.EncryptedData, &secret.Meta, &secret.Version,
			&secret.CreatedAt, &updatedAt, &deletedAt,
		)
		if err != nil {
			return nil, err
		}

		if updatedAt.Valid {
			secret.UpdatedAt = &updatedAt.Time
		}
		if deletedAt.Valid {
			secret.DeletedAt = &deletedAt.Time
		}

		secrets = append(secrets, secret)
	}

	return secrets, rows.Err()
}
