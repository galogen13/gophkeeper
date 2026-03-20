package crypto

import (
	"os"
	"path/filepath"
)

// Keychain управляет сохранением мастер-ключа
type Keychain struct {
	keyDir string
}

func NewKeychain() (*Keychain, error) {
	// Определяем директорию для ключей
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, err
	}

	keyDir := filepath.Join(home, ".gophkeeper")
	if err := os.MkdirAll(keyDir, 0700); err != nil {
		return nil, err
	}

	return &Keychain{keyDir: keyDir}, nil
}

// SaveSalt сохраняет соль для мастер-ключа
func (k *Keychain) SaveSalt(salt []byte) error {
	saltPath := filepath.Join(k.keyDir, "salt")
	return os.WriteFile(saltPath, salt, 0600)
}

// LoadSalt загружает соль
func (k *Keychain) LoadSalt() ([]byte, error) {
	saltPath := filepath.Join(k.keyDir, "salt")
	salt, err := os.ReadFile(saltPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil // соли ещё нет
		}
		return nil, err
	}
	return salt, nil
}

// SaveToken сохраняет токен доступа (в открытом виде, т.к. он кратковременный)
func (k *Keychain) SaveToken(token string) error {
	tokenPath := filepath.Join(k.keyDir, "token")
	return os.WriteFile(tokenPath, []byte(token), 0600)
}

// LoadToken загружает токен
func (k *Keychain) LoadToken() (string, error) {
	tokenPath := filepath.Join(k.keyDir, "token")
	token, err := os.ReadFile(tokenPath)
	if err != nil {
		if os.IsNotExist(err) {
			return "", nil
		}
		return "", err
	}
	return string(token), nil
}

// Clear очищает все данные
func (k *Keychain) Clear() error {
	return os.RemoveAll(k.keyDir)
}
