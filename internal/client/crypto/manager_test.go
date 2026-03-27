package crypto_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/galogen13/gophkeeper/internal/client/crypto"
)

func TestMasterKeyManager_New(t *testing.T) {
	t.Skip("Requires mocking home directory or adding test constructor")

	manager, err := crypto.NewMasterKeyManager()
	if err != nil {
		t.Skip("Skipping: cannot create manager in test environment")
	}
	assert.NotNil(t, manager)
}

func TestMasterKeyManager_InitAndLoad(t *testing.T) {
	tempDir := t.TempDir()
	keychain := crypto.NewKeychainForTest(tempDir)

	manager := &crypto.MasterKeyManager{}
	manager.SetKeychainForTest(keychain)

	masterPass := "my-secret-master-pass"

	err := manager.InitMasterKey(masterPass)
	require.NoError(t, err)
	assert.True(t, manager.HasMasterKey())
	assert.NotNil(t, manager.GetMasterKey())

	salt, err := keychain.LoadSalt()
	require.NoError(t, err)
	assert.NotNil(t, salt)
	assert.Len(t, salt, 16)

	testPath := filepath.Join(tempDir, "test")
	_, err = os.Stat(testPath)
	assert.NoError(t, err)

	manager2 := &crypto.MasterKeyManager{}
	manager2.SetKeychainForTest(keychain)

	err = manager2.LoadMasterKey(masterPass)
	require.NoError(t, err)
	assert.NotNil(t, manager2.GetMasterKey())

	mk := manager2.GetMasterKey()
	original := "test пример"
	encrypted, err := mk.EncryptString(original)
	require.NoError(t, err)
	assert.NotEmpty(t, encrypted)

	decrypted, err := mk.DecryptString(encrypted)
	require.NoError(t, err)
	assert.Equal(t, original, decrypted)
}

func TestMasterKeyManager_Load_WrongPassword(t *testing.T) {
	manager := newTestManager(t)

	err := manager.InitMasterKey("correct-password")
	require.NoError(t, err)

	manager2 := newTestManager(t)
	manager2.SetKeychainForTest(manager.GetKeychain())

	err = manager2.LoadMasterKey("wrong-password")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid master password")
	assert.Nil(t, manager2.GetMasterKey())
}

func TestMasterKeyManager_Load_NotInitialized(t *testing.T) {
	manager := newTestManager(t)

	err := manager.LoadMasterKey("any-password")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not initialized")
}

func TestMasterKeyManager_Clear(t *testing.T) {
	manager := newTestManager(t)

	err := manager.InitMasterKey("test-pass")
	require.NoError(t, err)

	assert.NotNil(t, manager.GetMasterKey())

	manager.Clear()

	assert.Nil(t, manager.GetMasterKey())
}

func TestMasterKeyManager_HasMasterKey(t *testing.T) {
	manager := newTestManager(t)

	assert.False(t, manager.HasMasterKey())

	err := manager.InitMasterKey("test-pass")
	require.NoError(t, err)
	assert.True(t, manager.HasMasterKey())
}

func newTestManager(t *testing.T) *crypto.MasterKeyManager {
	tempDir := t.TempDir()
	keychain := crypto.NewKeychainForTest(tempDir)

	manager := &crypto.MasterKeyManager{}
	manager.SetKeychainForTest(keychain)
	return manager
}
