package crypto

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
	"errors"
	"fmt"
	"strings"

	"golang.org/x/crypto/argon2"
)

type PasswordConfig struct {
	Time       uint32
	Memory     uint32
	Threads    uint8
	KeyLength  uint32
	SaltLength uint32
}

var DefaultPasswordConfig = PasswordConfig{
	Time:       1,
	Memory:     64 * 1024, // 64 MB
	Threads:    4,
	KeyLength:  32,
	SaltLength: 16,
}

type PasswordHasher struct {
	config PasswordConfig
}

func NewPasswordHasher(config PasswordConfig) *PasswordHasher {
	return &PasswordHasher{config: config}
}

// HashPassword хеширует пароль с солью
func (h *PasswordHasher) HashPassword(password string) (string, error) {
	// Генерируем соль
	salt := make([]byte, h.config.SaltLength)
	_, err := rand.Read(salt)
	if err != nil {
		return "", err
	}

	// Хешируем пароль
	hash := argon2.IDKey(
		[]byte(password),
		salt,
		h.config.Time,
		h.config.Memory,
		h.config.Threads,
		h.config.KeyLength,
	)

	// Формат: $argon2id$v=19$m=65536,t=1,p=4$c2FsdA$hash
	b64Salt := base64.RawStdEncoding.EncodeToString(salt)
	b64Hash := base64.RawStdEncoding.EncodeToString(hash)

	encodedHash := fmt.Sprintf("$argon2id$v=19$m=%d,t=%d,p=%d$%s$%s",
		h.config.Memory, h.config.Time, h.config.Threads, b64Salt, b64Hash)

	return encodedHash, nil
}

// VerifyPassword проверяет пароль на соответствие хешу
func (h *PasswordHasher) VerifyPassword(password, encodedHash string) (bool, error) {
	// Разбираем формат
	parts := strings.Split(encodedHash, "$")
	if len(parts) != 6 {
		return false, errors.New("invalid hash format")
	}

	var memory, time uint32
	var threads uint8
	_, err := fmt.Sscanf(parts[3], "m=%d,t=%d,p=%d", &memory, &time, &threads)
	if err != nil {
		return false, err
	}

	salt, err := base64.RawStdEncoding.DecodeString(parts[4])
	if err != nil {
		return false, err
	}

	hash, err := base64.RawStdEncoding.DecodeString(parts[5])
	if err != nil {
		return false, err
	}

	// Вычисляем хеш для проверяемого пароля
	otherHash := argon2.IDKey([]byte(password), salt, time, memory, threads, uint32(len(hash)))

	// Сравниваем хеши
	if subtle.ConstantTimeCompare(hash, otherHash) == 1 {
		return true, nil
	}

	return false, nil
}
