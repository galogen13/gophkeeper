package crypto

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"io"
	"os"
	"path/filepath"

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

// MasterKeyManager управляет мастер-ключом и его хранением
type MasterKeyManager struct {
	keyDir     string
	masterKey  *MasterKey
	masterPass string
}

// NewMasterKeyManager создаёт менеджер мастер-ключа
func NewMasterKeyManager() (*MasterKeyManager, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, err
	}

	keyDir := filepath.Join(home, ".gophkeeper")
	if err := os.MkdirAll(keyDir, 0700); err != nil {
		return nil, err
	}

	return &MasterKeyManager{
		keyDir: keyDir,
	}, nil
}

// InitMasterKey инициализирует мастер-ключ по паролю (при первом входе)
func (m *MasterKeyManager) InitMasterKey(masterPass string) error {
	// Генерируем соль
	salt, err := GenerateSalt()
	if err != nil {
		return err
	}

	// Сохраняем соль
	saltPath := filepath.Join(m.keyDir, "salt")
	if err := os.WriteFile(saltPath, salt, 0600); err != nil {
		return err
	}

	// Создаём мастер-ключ
	m.masterKey = NewMasterKeyFromPassword(masterPass, salt)
	m.masterPass = masterPass

	return nil
}

// LoadMasterKey загружает мастер-ключ по паролю (при входе)
func (m *MasterKeyManager) LoadMasterKey(masterPass string) error {
	saltPath := filepath.Join(m.keyDir, "salt")
	salt, err := os.ReadFile(saltPath)
	if err != nil {
		if os.IsNotExist(err) {
			return errors.New("master key not initialized. Please register first")
		}
		return err
	}

	m.masterKey = NewMasterKeyFromPassword(masterPass, salt)
	m.masterPass = masterPass

	// Проверяем, что ключ работает (пробуем расшифровать тестовую строку)
	testPath := filepath.Join(m.keyDir, "test")
	if _, err := os.Stat(testPath); err == nil {
		testEncrypted, err := os.ReadFile(testPath)
		if err != nil {
			return err
		}
		_, err = m.masterKey.Decrypt(testEncrypted)
		if err != nil {
			return errors.New("invalid master password")
		}
	} else {
		// Сохраняем тестовую строку для проверки при следующем входе
		testData, err := m.masterKey.Encrypt([]byte("test"))
		if err != nil {
			return err
		}
		os.WriteFile(testPath, testData, 0600)
	}

	return nil
}

// GetMasterKey возвращает мастер-ключ (только для внутреннего использования)
func (m *MasterKeyManager) GetMasterKey() *MasterKey {
	return m.masterKey
}

// Clear очищает мастер-ключ из памяти (при выходе)
func (m *MasterKeyManager) Clear() {
	m.masterKey = nil
	m.masterPass = ""
}

// HasMasterKey проверяет, инициализирован ли мастер-ключ
func (m *MasterKeyManager) HasMasterKey() bool {
	saltPath := filepath.Join(m.keyDir, "salt")
	_, err := os.Stat(saltPath)
	return err == nil
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
