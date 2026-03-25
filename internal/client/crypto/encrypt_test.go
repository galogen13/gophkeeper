package crypto_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/galogen13/gophkeeper/internal/client/crypto"
)

func TestGenerateSalt(t *testing.T) {
	t.Parallel()

	salt1, err := crypto.GenerateSalt()
	require.NoError(t, err)
	assert.Len(t, salt1, 16)

	salt2, err := crypto.GenerateSalt()
	require.NoError(t, err)
	assert.Len(t, salt2, 16)

	assert.NotEqual(t, salt1, salt2)
}

func TestNewMasterKeyFromPassword(t *testing.T) {
	t.Parallel()

	salt := []byte("test-salt-12345678")
	mk := crypto.NewMasterKeyFromPassword("test-password", salt)
	assert.NotNil(t, mk)
}

func TestMasterKey_EncryptDecrypt(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		password  string
		salt      []byte
		plaintext []byte
		wantErr   bool
	}{
		{
			name:      "successful encrypt/decrypt",
			password:  "test-password",
			salt:      []byte("salt-12345678"),
			plaintext: []byte("secret data"),
			wantErr:   false,
		},
		{
			name:      "empty password",
			password:  "",
			salt:      []byte("salt-12345678"),
			plaintext: []byte("secret data"),
			wantErr:   false,
		},
		{
			name:      "empty plaintext",
			password:  "test-password",
			salt:      []byte("salt-12345678"),
			plaintext: []byte{},
			wantErr:   false,
		},
		{
			name:      "long plaintext (10KB)",
			password:  "test-password",
			salt:      []byte("salt-12345678"),
			plaintext: make([]byte, 10*1024),
			wantErr:   false,
		},
		{
			name:      "unicode plaintext",
			password:  "test-password",
			salt:      []byte("salt-12345678"),
			plaintext: []byte("привет мир!!"),
			wantErr:   false,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			mk := crypto.NewMasterKeyFromPassword(tt.password, tt.salt)
			require.NotNil(t, mk)

			encrypted, err := mk.Encrypt(tt.plaintext)
			if tt.wantErr {
				assert.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.NotEmpty(t, encrypted)

			decrypted, err := mk.Decrypt(encrypted)
			require.NoError(t, err)
			assert.Equal(t, tt.plaintext, decrypted)
		})
	}
}

func TestMasterKey_EncryptDecrypt_InvalidKey(t *testing.T) {
	t.Parallel()

	mk1 := crypto.NewMasterKeyFromPassword("correct-pass", []byte("salt-12345678"))
	encrypted, err := mk1.Encrypt([]byte("secret"))
	require.NoError(t, err)

	mk2 := crypto.NewMasterKeyFromPassword("wrong-pass", []byte("salt-12345678"))
	_, err = mk2.Decrypt(encrypted)
	assert.Error(t, err, "decrypting with wrong key should fail")
}

func TestMasterKey_EncryptDecrypt_DifferentSalt(t *testing.T) {
	t.Parallel()

	password := "test-password"
	plaintext := []byte("secret")

	mk1 := crypto.NewMasterKeyFromPassword(password, []byte("salt-12345678"))
	encrypted1, err := mk1.Encrypt(plaintext)
	require.NoError(t, err)

	mk2 := crypto.NewMasterKeyFromPassword(password, []byte("different-salt"))
	encrypted2, err := mk2.Encrypt(plaintext)
	require.NoError(t, err)

	assert.NotEqual(t, encrypted1, encrypted2)

	_, err = mk1.Decrypt(encrypted2)
	assert.Error(t, err)

	_, err = mk2.Decrypt(encrypted1)
	assert.Error(t, err)
}

func TestMasterKey_EncryptString_DecryptString(t *testing.T) {
	t.Parallel()

	mk := crypto.NewMasterKeyFromPassword("test-pass", []byte("salt-12345678"))

	tests := []struct {
		name      string
		plaintext string
	}{
		{"simple string", "hello world"},
		{"with special chars", "P@ssw0rd!@#$%^&*()"},
		{"unicode", "пароль 密码 password"},
		{"long string", string(make([]byte, 10000))},
		{"empty", ""},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			encrypted, err := mk.EncryptString(tt.plaintext)
			require.NoError(t, err)
			assert.NotEmpty(t, encrypted)

			decrypted, err := mk.DecryptString(encrypted)
			require.NoError(t, err)
			assert.Equal(t, tt.plaintext, decrypted)
		})
	}
}

func TestMasterKey_NilKey(t *testing.T) {
	t.Parallel()

	var mk *crypto.MasterKey

	_, err := mk.Encrypt([]byte("data"))
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not initialized")

	_, err = mk.Decrypt([]byte("data"))
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not initialized")
}

func TestMasterKey_EmptyKey(t *testing.T) {
	mk := crypto.NewMasterKeyFromPassword("", []byte("salt-12345678"))
	assert.NotNil(t, mk)

	encrypted, err := mk.Encrypt([]byte("test"))
	require.NoError(t, err)

	decrypted, err := mk.Decrypt(encrypted)
	require.NoError(t, err)
	assert.Equal(t, []byte("test"), decrypted)
}
