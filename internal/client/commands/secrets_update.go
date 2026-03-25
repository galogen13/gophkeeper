package commands

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"github.com/galogen13/gophkeeper/internal/client"
	"github.com/galogen13/gophkeeper/internal/client/grpc"
	"github.com/spf13/cobra"
	"golang.org/x/term"
)

func NewSecretsUpdateCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "update [id]",
		Short: "Update an existing secret",
		Long: `Update an existing secret. You will be prompted to enter new values.
Leave fields empty to keep current values.`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return updateSecret(args[0])
		},
	}

	return cmd
}

func updateSecret(id string) error {
	ctx := context.Background()

	// Проверяем аутентификацию
	auth, err := store.GetAuth(ctx)
	if err != nil {
		return fmt.Errorf("failed to get auth: %w", err)
	}
	if auth == nil {
		return fmt.Errorf("not logged in. Please run 'gophkeeper auth login' first")
	}

	mk := keyManager.GetMasterKey()
	if mk == nil {
		return fmt.Errorf("master key not loaded. Please login again")
	}

	// Получаем существующий секрет из локального хранилища
	secret, err := store.GetSecret(ctx, id)
	if err != nil {
		return fmt.Errorf("failed to get secret: %w", err)
	}

	fmt.Printf("Updating secret: %s (type: %s)\n", secret.Title, secret.Type.String())
	fmt.Println("Leave field empty to keep current value")
	fmt.Println()

	// Обновляем общие поля
	newTitle := promptString("Title", secret.Title)
	if newTitle != "" {
		secret.Title = newTitle
	}

	newMeta := promptString("Meta (JSON)", secret.Meta)
	if newMeta != "" {
		secret.Meta = newMeta
	}

	// Обновляем данные в зависимости от типа секрета
	var updatedData []byte
	switch secret.Type {
	case client.SecretTypeCredentials:
		updatedData, err = updateCredentialsData(secret.EncryptedData)
	case client.SecretTypeBankCard:
		updatedData, err = updateBankCardData(secret.EncryptedData)
	case client.SecretTypeText:
		updatedData, err = updateTextData(secret.EncryptedData)
	case client.SecretTypeBinary:
		updatedData, err = updateBinaryData(secret.EncryptedData)
	default:
		return fmt.Errorf("unknown secret type: %v", secret.Type)
	}

	if err != nil {
		return err
	}

	// Если данные были обновлены, заменяем
	if updatedData != nil {

		encryptedData, err := mk.Encrypt(updatedData)
		if err != nil {
			return fmt.Errorf("failed to encrypt data: %w", err)
		}
		secret.EncryptedData = encryptedData
	}

	// Обновляем время изменения
	now := time.Now()
	secret.UpdatedAt = &now

	// Сохраняем локально
	if err := store.SaveSecrets(ctx, []*client.Secret{secret}); err != nil {
		return fmt.Errorf("failed to update secret locally: %w", err)
	}

	// Отправляем на сервер
	authCtx := grpcClient.WithAuth(ctx)
	_, err = grpcClient.GetKeeperClient().UpdateSecret(authCtx, grpc.SecretToProtoSecretUpdate(secret))
	if err != nil {
		return fmt.Errorf("failed to update secret on server: %w", err)
	}

	fmt.Printf("\nSecret %s updated successfully\n", id)

	return nil
}

// updateCredentialsData обновляет данные типа "логин/пароль"
func updateCredentialsData(currentData []byte) ([]byte, error) {
	var data client.CredentialsData

	// Расшифровываем текущие данные
	if len(currentData) > 0 {
		if err := json.Unmarshal(currentData, &data); err != nil {
			return nil, fmt.Errorf("failed to parse current data: %w", err)
		}
	}

	fmt.Println("\n--- Credentials ---")

	login := promptString("Login", data.Login)
	if login != "" {
		data.Login = login
	}

	password := promptPassword("Password")
	if password != "" {
		data.Password = password
	}

	url := promptString("URL", data.URL)
	if url != "" {
		data.URL = url
	}

	newData, err := json.Marshal(data)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal data: %w", err)
	}

	return newData, nil
}

// updateBankCardData обновляет данные банковской карты
func updateBankCardData(currentData []byte) ([]byte, error) {
	var data client.BankCardData

	if len(currentData) > 0 {
		if err := json.Unmarshal(currentData, &data); err != nil {
			return nil, fmt.Errorf("failed to parse current data: %w", err)
		}
	}

	fmt.Println("\n--- Bank Card ---")

	cardNumber := promptString("Card Number", data.CardNumber)
	if cardNumber != "" {
		data.CardNumber = cardNumber
	}

	cardHolder := promptString("Card Holder", data.CardHolder)
	if cardHolder != "" {
		data.CardHolder = cardHolder
	}

	expiry := promptString("Expiry (MM/YY)", fmt.Sprintf("%s/%s", data.ExpiryMonth, data.ExpiryYear))
	if expiry != "" {
		parts := strings.Split(expiry, "/")
		if len(parts) == 2 {
			data.ExpiryMonth = parts[0]
			data.ExpiryYear = parts[1]
		}
	}

	cvv := promptString("CVV", data.CVV)
	if cvv != "" {
		data.CVV = cvv
	}

	bankName := promptString("Bank Name", data.BankName)
	if bankName != "" {
		data.BankName = bankName
	}

	newData, err := json.Marshal(data)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal data: %w", err)
	}

	return newData, nil
}

// updateTextData обновляет текстовые данные
func updateTextData(currentData []byte) ([]byte, error) {
	var data client.TextData

	if len(currentData) > 0 {
		if err := json.Unmarshal(currentData, &data); err != nil {
			return nil, fmt.Errorf("failed to parse current data: %w", err)
		}
	}

	fmt.Println("\n--- Text Content ---")
	fmt.Println("Current content:")
	fmt.Println("---")
	fmt.Println(data.Content)
	fmt.Println("---")

	fmt.Print("Enter new content (Ctrl+D to finish, Enter to keep current): ")

	var lines []string
	scanner := bufio.NewScanner(os.Stdin)
	for scanner.Scan() {
		line := scanner.Text()
		lines = append(lines, line)
	}

	if len(lines) > 0 {
		newContent := strings.Join(lines, "\n")
		if newContent != "" {
			data.Content = newContent
		}
	}

	newData, err := json.Marshal(data)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal data: %w", err)
	}

	return newData, nil
}

// updateBinaryData обновляет бинарные данные
func updateBinaryData(currentData []byte) ([]byte, error) {
	var data client.BinaryData

	if len(currentData) > 0 {
		if err := json.Unmarshal(currentData, &data); err != nil {
			return nil, fmt.Errorf("failed to parse current data: %w", err)
		}
	}

	fmt.Println("\n--- Binary File ---")
	fmt.Printf("Current file: %s (%d bytes)\n", data.Filename, data.Size)

	fmt.Print("Enter path to new file (or press Enter to keep current): ")
	var filePath string
	fmt.Scanln(&filePath)

	if filePath != "" {
		fileData, err := os.ReadFile(filePath)
		if err != nil {
			return nil, fmt.Errorf("failed to read file: %w", err)
		}

		data.Filename = filepath.Base(filePath)
		data.Size = int64(len(fileData))
		data.ContentType = "application/octet-stream"

		return fileData, nil
	}

	return currentData, nil
}

// promptString запрашивает строку с подсказкой
func promptString(prompt, current string) string {
	fmt.Printf("%s [%s]: ", prompt, current)
	scanner := bufio.NewScanner(os.Stdin)
	if scanner.Scan() {
		input := strings.TrimSpace(scanner.Text())
		if input != "" {
			return input
		}
	}
	return ""
}

// promptPassword запрашивает пароль (скрытый ввод)
func promptPassword(prompt string) string {
	fmt.Printf("%s: ", prompt)
	password, err := term.ReadPassword(int(syscall.Stdin))
	fmt.Println()
	if err != nil {
		return ""
	}
	return string(password)
}
