package crypto_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/galogen13/gophkeeper/internal/server/crypto"
)

func TestPasswordHasher_HashAndVerify(t *testing.T) {
	t.Parallel()

	hasher := crypto.NewPasswordHasher(crypto.DefaultPasswordConfig)

	tests := []struct {
		name     string
		password string
		wantErr  bool
	}{
		{"simple password", "password123", false},
		{"with special chars", "P@ssw0rd!@#$", false},
		{"long password", string(make([]byte, 100)), false},
		{"empty password", "", false},
		{"unicode password", "пароль", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			hash, err := hasher.HashPassword(tt.password)
			if tt.wantErr {
				assert.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.NotEmpty(t, hash)

			// Проверяем правильный пароль
			valid, err := hasher.VerifyPassword(tt.password, hash)
			require.NoError(t, err)
			assert.True(t, valid)

			// Проверяем неправильный пароль
			valid, err = hasher.VerifyPassword("wrong-password", hash)
			require.NoError(t, err)
			assert.False(t, valid)
		})
	}
}

func TestPasswordHasher_InvalidHash(t *testing.T) {
	t.Parallel()

	hasher := crypto.NewPasswordHasher(crypto.DefaultPasswordConfig)

	_, err := hasher.VerifyPassword("password", "invalid-hash")
	assert.Error(t, err)

	_, err = hasher.VerifyPassword("password", "")
	assert.Error(t, err)
}

func TestPasswordHasher_DifferentSalts(t *testing.T) {
	t.Parallel()

	hasher := crypto.NewPasswordHasher(crypto.DefaultPasswordConfig)
	password := "test-password"

	hash1, err := hasher.HashPassword(password)
	require.NoError(t, err)

	hash2, err := hasher.HashPassword(password)
	require.NoError(t, err)

	// Хеши должны отличаться из-за разных солей
	assert.NotEqual(t, hash1, hash2)

	// Но оба должны проверяться правильно
	valid, err := hasher.VerifyPassword(password, hash1)
	require.NoError(t, err)
	assert.True(t, valid)

	valid, err = hasher.VerifyPassword(password, hash2)
	require.NoError(t, err)
	assert.True(t, valid)
}
