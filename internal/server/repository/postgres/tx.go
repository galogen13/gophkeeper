package postgres

import (
	"context"
	"database/sql"
)

type transactionManager struct {
	db *sql.DB
}

func NewTransactionManager(db *sql.DB) *transactionManager {
	return &transactionManager{db: db}
}

func (tm *transactionManager) WithinTransaction(ctx context.Context, fn func(ctx context.Context) error) error {
	// Начинаем транзакцию
	tx, err := tm.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}

	// Сохраняем транзакцию в контекст
	txCtx := context.WithValue(ctx, txKey{}, tx)

	// Выполняем функцию
	err = fn(txCtx)
	if err != nil {
		// Ошибка - откатываем транзакцию
		if rbErr := tx.Rollback(); rbErr != nil {
			return rbErr
		}
		return err
	}

	// Всё хорошо - коммитим
	return tx.Commit()
}

type txKey struct{}

// GetTx извлекает транзакцию из контекста (для использования в репозиториях)
func GetTx(ctx context.Context) (*sql.Tx, bool) {
	tx, ok := ctx.Value(txKey{}).(*sql.Tx)
	return tx, ok
}
