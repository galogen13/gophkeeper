package postgres

import (
	"database/sql"
	"time"

	"github.com/galogen13/gophkeeper/internal/logger"
	"github.com/galogen13/gophkeeper/internal/server/config"
)

func ConnectDB(cfg config.DatabaseConfig) (*sql.DB, error) {
	connStr := cfg.ConnString()
	db, err := sql.Open("postgres", connStr)
	if err != nil {
		return nil, err
	}

	db.SetMaxOpenConns(25)
	db.SetMaxIdleConns(5)
	db.SetConnMaxLifetime(5 * time.Minute)

	if err := db.Ping(); err != nil {
		return nil, err
	}

	logger.Log.Info("Connected to database")
	return db, nil
}
