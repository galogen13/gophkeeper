-- Создаём расширение для UUID (если ещё не создано)
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

-- Таблица пользователей
CREATE TABLE users (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    email VARCHAR(255) UNIQUE NOT NULL,
    password_hash BYTEA NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE
);

-- Таблица секретов
CREATE TABLE secrets (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    owner_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    type SMALLINT NOT NULL,
    title VARCHAR(255) NOT NULL,
    encrypted_data BYTEA NOT NULL,
    meta TEXT,  -- JSON метаинформация
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE,
    deleted_at TIMESTAMP WITH TIME ZONE,  -- soft delete
    UNIQUE(owner_id, id)
);

-- Индексы для быстрого поиска
CREATE INDEX idx_secrets_owner_id ON secrets(owner_id);
CREATE INDEX idx_secrets_updated_at ON secrets(updated_at) WHERE deleted_at IS NULL;
CREATE INDEX idx_secrets_type ON secrets(type) WHERE deleted_at IS NULL;