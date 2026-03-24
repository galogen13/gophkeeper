package server

import (
	"time"

	"github.com/google/uuid"
)

type SecretType int

const (
	SecretTypeUnknown SecretType = iota
	SecretTypeCredentials
	SecretTypeText
	SecretTypeBinary
	SecretTypeBankCard
)

func (s SecretType) String() string {
	switch s {
	case SecretTypeCredentials:
		return "credentials"
	case SecretTypeText:
		return "text"
	case SecretTypeBinary:
		return "binary"
	case SecretTypeBankCard:
		return "bank_card"
	default:
		return "unknown"
	}
}

type Secret struct {
	ID            uuid.UUID  `json:"id"`
	OwnerID       uuid.UUID  `json:"owner_id"`
	Type          SecretType `json:"type"`
	Title         string     `json:"title"`
	EncryptedData []byte     `json:"encrypted_data"`
	Meta          string     `json:"meta"` // JSON строка с метаинформацией
	CreatedAt     time.Time  `json:"created_at"`
	UpdatedAt     *time.Time `json:"updated_at,omitempty"`
	DeletedAt     *time.Time `json:"deleted_at,omitempty"` // soft delete
}

type CredentialsData struct {
	Login    string `json:"login"`
	Password string `json:"password"`
	URL      string `json:"url,omitempty"`
}

type BankCardData struct {
	CardNumber  string `json:"card_number"`
	CardHolder  string `json:"card_holder"`
	ExpiryMonth string `json:"expiry_month"`
	ExpiryYear  string `json:"expiry_year"`
	CVV         string `json:"cvv,omitempty"`
	BankName    string `json:"bank_name,omitempty"`
}

type TextData struct {
	Content string `json:"content"`
}

type BinaryData struct {
	Filename    string `json:"filename"`
	ContentType string `json:"content_type"`
	Size        int64  `json:"size"`
}
