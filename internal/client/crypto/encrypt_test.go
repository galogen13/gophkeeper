package crypto_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/galogen13/gophkeeper/internal/client/crypto"
)

func TestMasterKey_EncryptDecrypt(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		password  string
		salt      []byte
		plaintext string
		wantErr   bool
	}{
		{
			name:      "successful encrypt/decrypt",
			password:  "test-password",
			salt:      []byte("test-salt-123456"),
			plaintext: "secret data",
			wantErr:   false,
		},
		{
			name:      "empty password",
			password:  "",
			salt:      []byte("test-salt-123456"),
			plaintext: "secret data",
			wantErr:   false, // пустой пароль тоже работает (но не рекомендуется)
		},
		{
			name:      "empty plaintext",
			password:  "test-password",
			salt:      []byte("test-salt-123456"),
			plaintext: "",
			wantErr:   false,
		},
		{
			name:      "long plaintext",
			password:  "test-password",
			salt:      []byte("test-salt-123456"),
			plaintext: string(make([]byte, 10000)),
			wantErr:   false,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			mk := crypto.NewMasterKeyFromPassword(tt.password, tt.salt)
			require.NotNil(t, mk)

			encrypted, err := mk.Encrypt([]byte(tt.plaintext))
			if tt.wantErr {
				assert.Error(t, err)
				return
			}
			assert.NoError(t, err)
			assert.NotEmpty(t, encrypted)

			decrypted, err := mk.Decrypt(encrypted)
			assert.NoError(t, err)
			assert.Equal(t, tt.plaintext, string(decrypted))
		})
	}
}

func TestMasterKey_EncryptString_DecryptString(t *testing.T) {
	t.Parallel()

	mk := crypto.NewMasterKeyFromPassword("test-pass", []byte("salt-12345678"))

	original := "my secret string"

	encrypted, err := mk.EncryptString(original)
	require.NoError(t, err)
	assert.NotEmpty(t, encrypted)

	decrypted, err := mk.DecryptString(encrypted)
	require.NoError(t, err)
	assert.Equal(t, original, decrypted)
}

func TestMasterKey_WrongPassword(t *testing.T) {
	t.Parallel()

	mk1 := crypto.NewMasterKeyFromPassword("correct-pass", []byte("salt-12345678"))
	encrypted, err := mk1.EncryptString("secret")
	require.NoError(t, err)

	mk2 := crypto.NewMasterKeyFromPassword("wrong-pass", []byte("salt-12345678"))

	_, err = mk2.DecryptString(encrypted)
	assert.Error(t, err, "decrypting with wrong password should fail")
}

func TestGenerateSalt(t *testing.T) {
	t.Parallel()

	salt1, err := crypto.GenerateSalt()
	require.NoError(t, err)
	assert.Len(t, salt1, 16)

	salt2, err := crypto.GenerateSalt()
	require.NoError(t, err)

	// Соли должны быть разными
	assert.NotEqual(t, salt1, salt2)
}
