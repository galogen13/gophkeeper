package crypto_test

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/galogen13/gophkeeper/internal/server/crypto"
)

func TestJWTManager_GenerateAndVerify(t *testing.T) {
	t.Parallel()

	manager := crypto.NewJWTManager(
		"test-secret-key-1234567890",
		15*time.Minute,
		7*24*time.Hour,
	)

	userID := uuid.New()

	tokenPair, err := manager.GenerateTokenPair(userID)
	require.NoError(t, err)
	assert.NotEmpty(t, tokenPair.AccessToken)
	assert.NotEmpty(t, tokenPair.RefreshToken)
	assert.Equal(t, 15*time.Minute, tokenPair.ExpiresIn)

	// Проверяем access token
	claims, err := manager.VerifyToken(tokenPair.AccessToken)
	require.NoError(t, err)
	assert.Equal(t, userID, claims.UserID)

	// Проверяем refresh token
	claims, err = manager.VerifyToken(tokenPair.RefreshToken)
	require.NoError(t, err)
	assert.Equal(t, userID, claims.UserID)
}

func TestJWTManager_ExpiredToken(t *testing.T) {
	t.Parallel()

	manager := crypto.NewJWTManager(
		"test-secret",
		-1*time.Minute, // уже истёкший
		7*24*time.Hour,
	)

	tokenPair, err := manager.GenerateTokenPair(uuid.New())
	require.NoError(t, err)

	_, err = manager.VerifyToken(tokenPair.AccessToken)
	assert.Error(t, err, "expired token should be invalid")
}

func TestJWTManager_WrongSecret(t *testing.T) {
	t.Parallel()

	manager1 := crypto.NewJWTManager("secret1", 15*time.Minute, 7*24*time.Hour)
	tokenPair, err := manager1.GenerateTokenPair(uuid.New())
	require.NoError(t, err)

	manager2 := crypto.NewJWTManager("secret2", 15*time.Minute, 7*24*time.Hour)

	_, err = manager2.VerifyToken(tokenPair.AccessToken)
	assert.Error(t, err, "token signed with wrong secret should be invalid")
}

func TestJWTManager_InvalidToken(t *testing.T) {
	t.Parallel()

	manager := crypto.NewJWTManager("test-secret", 15*time.Minute, 7*24*time.Hour)

	_, err := manager.VerifyToken("invalid-token-format")
	assert.Error(t, err)

	_, err = manager.VerifyToken("")
	assert.Error(t, err)
}
