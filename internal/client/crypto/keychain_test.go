package crypto_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/galogen13/gophkeeper/internal/client/crypto"
)

func TestKeychain_NewKeychain(t *testing.T) {
	tempDir := t.TempDir()
	keychain := crypto.NewKeychainForTest(tempDir)

	assert.NotNil(t, keychain)
	assert.Equal(t, tempDir, keychain.GetKeyDir())

	info, err := os.Stat(tempDir)
	require.NoError(t, err)
	assert.True(t, info.IsDir())
}

func TestKeychain_SaveAndLoadSalt(t *testing.T) {
	tempDir := t.TempDir()
	keychain := crypto.NewKeychainForTest(tempDir)

	salt := []byte("test-salt-12345678")

	err := keychain.SaveSalt(salt)
	require.NoError(t, err)

	saltPath := filepath.Join(tempDir, "salt")
	_, err = os.Stat(saltPath)
	require.NoError(t, err)

	loaded, err := keychain.LoadSalt()
	require.NoError(t, err)
	assert.Equal(t, salt, loaded)
}

func TestKeychain_LoadSalt_NotExist(t *testing.T) {
	tempDir := t.TempDir()
	keychain := crypto.NewKeychainForTest(tempDir)

	salt, err := keychain.LoadSalt()
	require.NoError(t, err)
	assert.Nil(t, salt)
}

func TestKeychain_LoadSalt_EmptyFile(t *testing.T) {
	tempDir := t.TempDir()
	keychain := crypto.NewKeychainForTest(tempDir)

	saltPath := filepath.Join(tempDir, "salt")
	err := os.WriteFile(saltPath, []byte{}, 0600)
	require.NoError(t, err)

	salt, err := keychain.LoadSalt()
	require.NoError(t, err)
	assert.Empty(t, salt)
}

func TestKeychain_SaveAndLoadToken(t *testing.T) {
	tempDir := t.TempDir()
	keychain := crypto.NewKeychainForTest(tempDir)

	token := "test-access-token-12345"

	err := keychain.SaveToken(token)
	require.NoError(t, err)

	tokenPath := filepath.Join(tempDir, "token")
	_, err = os.Stat(tokenPath)
	require.NoError(t, err)

	loaded, err := keychain.LoadToken()
	require.NoError(t, err)
	assert.Equal(t, token, loaded)
}

func TestKeychain_LoadToken_NotExist(t *testing.T) {
	tempDir := t.TempDir()
	keychain := crypto.NewKeychainForTest(tempDir)

	token, err := keychain.LoadToken()
	require.NoError(t, err)
	assert.Equal(t, "", token)
}

func TestKeychain_Clear(t *testing.T) {
	tempDir := t.TempDir()
	keychain := crypto.NewKeychainForTest(tempDir)

	require.NoError(t, keychain.SaveSalt([]byte("salt")))
	require.NoError(t, keychain.SaveToken("token"))

	_, err := os.Stat(tempDir)
	require.NoError(t, err)

	err = keychain.Clear()
	require.NoError(t, err)

	_, err = os.Stat(tempDir)
	assert.True(t, os.IsNotExist(err))
}

func TestKeychain_Clear_EmptyDir(t *testing.T) {
	tempDir := t.TempDir()
	keychain := crypto.NewKeychainForTest(tempDir)

	err := keychain.Clear()
	require.NoError(t, err)

	_, err = os.Stat(tempDir)
	assert.True(t, os.IsNotExist(err))
}

func TestKeychain_SaveSalt_Permissions(t *testing.T) {
	tempDir := t.TempDir()
	keychain := crypto.NewKeychainForTest(tempDir)

	salt := []byte("test-salt")

	err := keychain.SaveSalt(salt)
	require.NoError(t, err)

	saltPath := filepath.Join(tempDir, "salt")
	_, err = os.Stat(saltPath)
	require.NoError(t, err)

}

func TestKeychain_SaveToken_Permissions(t *testing.T) {
	tempDir := t.TempDir()
	keychain := crypto.NewKeychainForTest(tempDir)

	err := keychain.SaveToken("token")
	require.NoError(t, err)

	tokenPath := filepath.Join(tempDir, "token")
	_, err = os.Stat(tokenPath)
	require.NoError(t, err)

}
