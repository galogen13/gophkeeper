package crypto

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"io"

	"golang.org/x/crypto/pbkdf2"
)

const (
	keyLength  = 32 // 256 бит
	saltLength = 16
	pbkdf2Iter = 100000
)

// MasterKey хранит ключ шифрования (только в памяти)
type MasterKey struct {
	key []byte
}

// NewMasterKeyFromPassword создаёт мастер-ключ из пароля и соли
func NewMasterKeyFromPassword(password string, salt []byte) *MasterKey {
	key := pbkdf2.Key([]byte(password), salt, pbkdf2Iter, keyLength, sha256.New)
	return &MasterKey{key: key}
}

// GenerateSalt генерирует случайную соль
func GenerateSalt() ([]byte, error) {
	salt := make([]byte, saltLength)
	_, err := rand.Read(salt)
	if err != nil {
		return nil, err
	}
	return salt, nil
}

// Encrypt шифрует данные с использованием AES-GCM
func (mk *MasterKey) Encrypt(plaintext []byte) ([]byte, error) {
	if mk == nil || len(mk.key) == 0 {
		return nil, errors.New("master key not initialized")
	}

	block, err := aes.NewCipher(mk.key)
	if err != nil {
		return nil, err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}

	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, err
	}

	ciphertext := gcm.Seal(nonce, nonce, plaintext, nil)
	return ciphertext, nil
}

// Decrypt расшифровывает данные
func (mk *MasterKey) Decrypt(ciphertext []byte) ([]byte, error) {
	if mk == nil || len(mk.key) == 0 {
		return nil, errors.New("master key not initialized")
	}

	block, err := aes.NewCipher(mk.key)
	if err != nil {
		return nil, err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}

	if len(ciphertext) < gcm.NonceSize() {
		return nil, errors.New("ciphertext too short")
	}

	nonce, ciphertext := ciphertext[:gcm.NonceSize()], ciphertext[gcm.NonceSize():]
	plaintext, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return nil, err
	}

	if plaintext == nil {
		return []byte{}, nil
	}

	return plaintext, nil
}

// EncryptString шифрует строку и возвращает base64
func (mk *MasterKey) EncryptString(plaintext string) (string, error) {
	encrypted, err := mk.Encrypt([]byte(plaintext))
	if err != nil {
		return "", err
	}
	return base64.StdEncoding.EncodeToString(encrypted), nil
}

// DecryptString расшифровывает base64 строку
func (mk *MasterKey) DecryptString(encrypted string) (string, error) {
	data, err := base64.StdEncoding.DecodeString(encrypted)
	if err != nil {
		return "", err
	}

	plaintext, err := mk.Decrypt(data)
	if err != nil {
		return "", err
	}

	return string(plaintext), nil
}
