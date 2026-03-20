package sqlite

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	_ "github.com/glebarez/sqlite"

	"github.com/galogen13/gophkeeper/internal/client"
	"github.com/galogen13/gophkeeper/internal/client/storage"
)

type SQLiteStorage struct {
	db *sql.DB
}

type Config struct {
	Path          string
	EncryptionKey []byte // ключ для шифрования SQLite (если используется SQLCipher)
}

func NewStorage(cfg Config) (*SQLiteStorage, error) {
	// Для обычного SQLite без шифрования
	db, err := sql.Open("sqlite", cfg.Path)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	// Включаем foreign keys и WAL режим для производительности
	if _, err := db.Exec("PRAGMA foreign_keys = ON"); err != nil {
		return nil, fmt.Errorf("failed to enable foreign keys: %w", err)
	}
	if _, err := db.Exec("PRAGMA journal_mode = WAL"); err != nil {
		return nil, fmt.Errorf("failed to set WAL mode: %w", err)
	}

	// Создаём таблицы
	if err := createTables(db); err != nil {
		return nil, fmt.Errorf("failed to create tables: %w", err)
	}

	return &SQLiteStorage{db: db}, nil
}

func createTables(db *sql.DB) error {
	queries := []string{
		// Таблица для хранения данных аутентификации (только одна запись)
		`CREATE TABLE IF NOT EXISTS auth (
            id INTEGER PRIMARY KEY CHECK (id = 1),
            user_id TEXT NOT NULL,
            access_token TEXT NOT NULL,
            refresh_token TEXT NOT NULL,
            expires_in INTEGER NOT NULL,
            saved_at TIMESTAMP NOT NULL
        )`,

		// Таблица для секретов
		`CREATE TABLE IF NOT EXISTS secrets (
            id TEXT PRIMARY KEY,
            type INTEGER NOT NULL,
            title TEXT NOT NULL,
            encrypted_data BLOB NOT NULL,
            meta TEXT,
            version INTEGER NOT NULL,
            created_at TIMESTAMP NOT NULL,
            updated_at TIMESTAMP,
            is_deleted BOOLEAN NOT NULL DEFAULT 0,
            deleted_at TIMESTAMP
        )`,

		// Индексы для быстрого поиска
		`CREATE INDEX IF NOT EXISTS idx_secrets_type ON secrets(type)`,
		`CREATE INDEX IF NOT EXISTS idx_secrets_is_deleted ON secrets(is_deleted)`,
		`CREATE INDEX IF NOT EXISTS idx_secrets_title ON secrets(title)`,

		// Таблица для метаданных синхронизации
		`CREATE TABLE IF NOT EXISTS sync_info (
            key TEXT PRIMARY KEY,
            value TEXT,
            updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
        )`,
	}

	for _, q := range queries {
		if _, err := db.Exec(q); err != nil {
			return fmt.Errorf("query failed: %s, error: %w", q, err)
		}
	}

	return nil
}

// Auth operations
func (s *SQLiteStorage) SaveAuth(ctx context.Context, auth *client.AuthInfo) error {
	query := `INSERT OR REPLACE INTO auth (id, user_id, access_token, refresh_token, expires_in, saved_at)
              VALUES (1, ?, ?, ?, ?, ?)`

	_, err := s.db.ExecContext(ctx, query,
		auth.UserID, auth.AccessToken, auth.RefreshToken, auth.ExpiresIn, auth.SavedAt,
	)
	return err
}

func (s *SQLiteStorage) GetAuth(ctx context.Context) (*client.AuthInfo, error) {
	query := `SELECT user_id, access_token, refresh_token, expires_in, saved_at FROM auth WHERE id = 1`

	var auth client.AuthInfo
	err := s.db.QueryRowContext(ctx, query).Scan(
		&auth.UserID, &auth.AccessToken, &auth.RefreshToken, &auth.ExpiresIn, &auth.SavedAt,
	)

	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil // нет сохранённой аутентификации
	}
	if err != nil {
		return nil, err
	}

	return &auth, nil
}

func (s *SQLiteStorage) ClearAuth(ctx context.Context) error {
	_, err := s.db.ExecContext(ctx, `DELETE FROM auth WHERE id = 1`)
	return err
}

// Secret operations
func (s *SQLiteStorage) SaveSecrets(ctx context.Context, secrets []*client.Secret) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	for _, secret := range secrets {
		query := `INSERT OR REPLACE INTO secrets 
                  (id, type, title, encrypted_data, meta, version, created_at, updated_at, is_deleted)
                  VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`

		_, err := tx.ExecContext(ctx, query,
			secret.ID, secret.Type, secret.Title, secret.EncryptedData,
			secret.Meta, secret.Version, secret.CreatedAt, secret.UpdatedAt, secret.IsDeleted,
		)
		if err != nil {
			return err
		}
	}

	return tx.Commit()
}

func (s *SQLiteStorage) GetSecret(ctx context.Context, id string) (*client.Secret, error) {
	query := `SELECT id, type, title, encrypted_data, meta, version, created_at, updated_at, is_deleted
              FROM secrets WHERE id = ?`

	var secret client.Secret
	var updatedAt sql.NullTime

	err := s.db.QueryRowContext(ctx, query, id).Scan(
		&secret.ID, &secret.Type, &secret.Title, &secret.EncryptedData,
		&secret.Meta, &secret.Version, &secret.CreatedAt, &updatedAt, &secret.IsDeleted,
	)

	if errors.Is(err, sql.ErrNoRows) {
		return nil, storage.ErrNotFound
	}
	if err != nil {
		return nil, err
	}

	if updatedAt.Valid {
		secret.UpdatedAt = &updatedAt.Time
	}

	return &secret, nil
}

func (s *SQLiteStorage) ListSecrets(ctx context.Context) ([]*client.Secret, error) {
	query := `SELECT id, type, title, encrypted_data, meta, version, created_at, updated_at, is_deleted
              FROM secrets ORDER BY created_at DESC`

	rows, err := s.db.QueryContext(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var secrets []*client.Secret
	for rows.Next() {
		var secret client.Secret
		var updatedAt sql.NullTime

		err := rows.Scan(
			&secret.ID, &secret.Type, &secret.Title, &secret.EncryptedData,
			&secret.Meta, &secret.Version, &secret.CreatedAt, &updatedAt, &secret.IsDeleted,
		)
		if err != nil {
			return nil, err
		}

		if updatedAt.Valid {
			secret.UpdatedAt = &updatedAt.Time
		}

		secrets = append(secrets, &secret)
	}

	return secrets, rows.Err()
}

func (s *SQLiteStorage) ListActiveSecrets(ctx context.Context) ([]*client.Secret, error) {
	query := `SELECT id, type, title, encrypted_data, meta, version, created_at, updated_at, is_deleted
              FROM secrets WHERE is_deleted = 0 ORDER BY created_at DESC`

	rows, err := s.db.QueryContext(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var secrets []*client.Secret
	for rows.Next() {
		var secret client.Secret
		var updatedAt sql.NullTime

		err := rows.Scan(
			&secret.ID, &secret.Type, &secret.Title, &secret.EncryptedData,
			&secret.Meta, &secret.Version, &secret.CreatedAt, &updatedAt, &secret.IsDeleted,
		)
		if err != nil {
			return nil, err
		}

		if updatedAt.Valid {
			secret.UpdatedAt = &updatedAt.Time
		}

		secrets = append(secrets, &secret)
	}

	return secrets, rows.Err()
}

func (s *SQLiteStorage) DeleteSecret(ctx context.Context, id string) error {
	// Soft delete
	query := `UPDATE secrets SET is_deleted = 1, deleted_at = ? WHERE id = ?`
	result, err := s.db.ExecContext(ctx, query, time.Now(), id)
	if err != nil {
		return err
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		return storage.ErrNotFound
	}

	return nil
}

// func (s *SQLiteStorage) UpdateSecret(ctx context.Context, secret *client.Secret) error {
// 	query := `UPDATE secrets
//               SET title = ?, encrypted_data = ?, meta = ?, version = ?, updated_at = ?, is_deleted = ?
//               WHERE id = ?`

// 	result, err := s.db.ExecContext(ctx, query,
// 		secret.Title, secret.EncryptedData, secret.Meta,
// 		secret.Version, secret.UpdatedAt, secret.IsDeleted, secret.ID,
// 	)
// 	if err != nil {
// 		return err
// 	}

// 	rows, err := result.RowsAffected()
// 	if err != nil {
// 		return err
// 	}
// 	if rows == 0 {
// 		return storage.ErrNotFound
// 	}

// 	return nil
// }

// Sync operations
func (s *SQLiteStorage) GetLastSyncTime(ctx context.Context) (*time.Time, error) {
	query := `SELECT value FROM sync_info WHERE key = 'last_sync'`

	var value string
	err := s.db.QueryRowContext(ctx, query).Scan(&value)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil // нет сохранённого времени
	}
	if err != nil {
		return nil, err
	}

	t, err := time.Parse(time.RFC3339, value)
	if err != nil {
		return nil, err
	}

	return &t, nil
}

func (s *SQLiteStorage) SetLastSyncTime(ctx context.Context, t time.Time) error {
	query := `INSERT OR REPLACE INTO sync_info (key, value, updated_at) VALUES ('last_sync', ?, ?)`
	_, err := s.db.ExecContext(ctx, query, t.Format(time.RFC3339), time.Now())
	return err
}

func (s *SQLiteStorage) Close() error {
	return s.db.Close()
}
