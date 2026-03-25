package crypto

import (
	"errors"
	"os"
	"path/filepath"
)

// MasterKeyManager управляет мастер-ключом в памяти
type MasterKeyManager struct {
	keychain  *Keychain
	masterKey *MasterKey
}

// NewMasterKeyManager создаёт новый менеджер мастер-ключа
func NewMasterKeyManager() (*MasterKeyManager, error) {
	keychain, err := NewKeychain()
	if err != nil {
		return nil, err
	}

	return &MasterKeyManager{
		keychain: keychain,
	}, nil
}

// InitMasterKey инициализирует мастер-ключ по паролю (при первом входе/регистрации)
func (m *MasterKeyManager) InitMasterKey(masterPass string) error {
	// Генерируем соль
	salt, err := GenerateSalt()
	if err != nil {
		return err
	}

	// Сохраняем соль
	if err := m.keychain.SaveSalt(salt); err != nil {
		return err
	}

	// Создаём мастер-ключ
	m.masterKey = NewMasterKeyFromPassword(masterPass, salt)

	// Сохраняем тестовую строку для проверки при следующем входе
	testData, err := m.masterKey.Encrypt([]byte("test"))
	if err != nil {
		return err
	}

	testPath := filepath.Join(m.keychain.keyDir, "test")
	if err := os.WriteFile(testPath, testData, 0600); err != nil {
		return err
	}

	return nil
}

// LoadMasterKey загружает мастер-ключ по паролю (при входе)
func (m *MasterKeyManager) LoadMasterKey(masterPass string) error {
	// Загружаем соль
	salt, err := m.keychain.LoadSalt()
	if err != nil {
		return err
	}
	if salt == nil {
		return errors.New("master key not initialized. Please register first")
	}

	// Создаём мастер-ключ
	testKey := NewMasterKeyFromPassword(masterPass, salt)

	// Проверяем, что ключ работает (декодируем тестовую строку)
	testPath := filepath.Join(m.keychain.keyDir, "test")
	testEncrypted, err := os.ReadFile(testPath)
	if err != nil {
		// Если тестового файла нет (первый вход после обновления), создаём его
		testData, err := testKey.Encrypt([]byte("test"))
		if err != nil {
			return err
		}
		if err := os.WriteFile(testPath, testData, 0600); err != nil {
			return err
		}
		// Сохраняем ключ только после успешной проверки
		m.masterKey = testKey
		return nil
	}

	// Пытаемся расшифровать тестовую строку
	_, err = testKey.Decrypt(testEncrypted)
	if err != nil {
		return errors.New("invalid master password")
	}

	m.masterKey = testKey

	return nil
}

// GetMasterKey возвращает мастер-ключ
func (m *MasterKeyManager) GetMasterKey() *MasterKey {
	return m.masterKey
}

// Clear очищает мастер-ключ из памяти
func (m *MasterKeyManager) Clear() {
	m.masterKey = nil
}

// HasMasterKey проверяет, инициализирован ли мастер-ключ (есть ли соль на диске)
func (m *MasterKeyManager) HasMasterKey() bool {
	salt, _ := m.keychain.LoadSalt()
	return salt != nil
}

// GetKeychain возвращает keychain (для тестов)
func (m *MasterKeyManager) GetKeychain() *Keychain {
	return m.keychain
}

// SetKeychainForTest устанавливает keychain для тестов (только для тестирования)
func (m *MasterKeyManager) SetKeychainForTest(kc *Keychain) {
	m.keychain = kc
}
