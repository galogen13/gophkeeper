package postgres

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/galogen13/gophkeeper/internal/server"
	"github.com/galogen13/gophkeeper/internal/server/repository"

	"github.com/google/uuid"
	"github.com/lib/pq"
)

type userRepository struct {
	db *sql.DB
}

func NewUserRepository(db *sql.DB) *userRepository {
	return &userRepository{db: db}
}

func (r *userRepository) Create(ctx context.Context, user *server.User) error {
	query := `
        INSERT INTO users (id, email, password_hash, created_at)
        VALUES ($1, $2, $3, $4)
    `

	if user.ID == uuid.Nil {
		user.ID = uuid.New()
	}

	_, err := r.db.ExecContext(ctx, query,
		user.ID, user.Email, user.PasswordHash, user.CreatedAt,
	)

	if err != nil {
		// Проверка на уникальность email
		if pqErr, ok := err.(*pq.Error); ok && pqErr.Code == "23505" {
			return repository.ErrEmailAlreadyExists
		}
		return err
	}

	return nil
}

func (r *userRepository) GetByEmail(ctx context.Context, email string) (*server.User, error) {
	query := `
        SELECT id, email, password_hash, created_at, updated_at
        FROM users
        WHERE email = $1
    `

	user := &server.User{}
	var updatedAt sql.NullTime

	err := r.db.QueryRowContext(ctx, query, email).Scan(
		&user.ID, &user.Email, &user.PasswordHash, &user.CreatedAt, &updatedAt,
	)

	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, repository.ErrUserNotFound
		}
		return nil, err
	}

	if updatedAt.Valid {
		user.UpdatedAt = updatedAt.Time
	}

	return user, nil
}

func (r *userRepository) GetByID(ctx context.Context, id uuid.UUID) (*server.User, error) {
	query := `
        SELECT id, email, password_hash, created_at, updated_at
        FROM users
        WHERE id = $1
    `

	user := &server.User{}
	var updatedAt sql.NullTime

	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&user.ID, &user.Email, &user.PasswordHash, &user.CreatedAt, &updatedAt,
	)

	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, repository.ErrUserNotFound
		}
		return nil, err
	}

	if updatedAt.Valid {
		user.UpdatedAt = updatedAt.Time
	}

	return user, nil
}

func (r *userRepository) Update(ctx context.Context, user *server.User) error {
	query := `
        UPDATE users
        SET email = $1, password_hash = $2, updated_at = $3
        WHERE id = $4
    `

	now := time.Now()
	result, err := r.db.ExecContext(ctx, query,
		user.Email, user.PasswordHash, now, user.ID,
	)

	if err != nil {
		return err
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return err
	}

	if rows == 0 {
		return repository.ErrUserNotFound
	}

	user.UpdatedAt = now
	return nil
}

func (r *userRepository) Delete(ctx context.Context, id uuid.UUID) error {
	// Здесь можно реализовать soft delete, но для пользователей обычно hard delete
	query := `DELETE FROM users WHERE id = $1`

	result, err := r.db.ExecContext(ctx, query, id)
	if err != nil {
		return err
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return err
	}

	if rows == 0 {
		return repository.ErrUserNotFound
	}

	return nil
}
