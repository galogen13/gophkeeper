package commands

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"syscall"

	"github.com/google/uuid"
	"github.com/spf13/cobra"
	"golang.org/x/term"

	"github.com/galogen13/gophkeeper/internal/client"
	"github.com/galogen13/gophkeeper/internal/client/grpc"
)

func NewSecretsCreateCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "create [type]",
		Short: "Create a new secret",
		Long: `Create a new secret of specified type.
Supported types: password, card, text, binary

Examples:
  gophkeeper secrets create password --title "GitHub" --login user --password pass123
  gophkeeper secrets create card --title "My Visa" --number 4111111111111111 --holder "IVANOV" --expiry 12/25
  gophkeeper secrets create text --title "Note" --content "Secret text"
  gophkeeper secrets create binary --title "Photo" --file ~/photo.jpg`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return createSecret(cmd, args[0])
		},
	}

	// Общие флаги
	cmd.Flags().String("title", "", "Secret title (required)")
	cmd.Flags().String("meta", "", "Additional metadata in JSON format")
	cmd.MarkFlagRequired("title")

	// Флаги для пароля
	cmd.Flags().String("login", "", "Login (for credentials type)")
	cmd.Flags().String("password", "", "Password (for credentials type)")
	cmd.Flags().String("url", "", "URL (for credentials type)")

	// Флаги для банковской карты
	cmd.Flags().String("number", "", "Card number")
	cmd.Flags().String("holder", "", "Card holder name")
	cmd.Flags().String("expiry", "", "Expiry date (MM/YY)")
	cmd.Flags().String("cvv", "", "CVV code")
	cmd.Flags().String("bank", "", "Bank name")

	// Флаги для текста
	cmd.Flags().String("content", "", "Text content")

	// Флаги для бинарных данных
	cmd.Flags().String("file", "", "Path to binary file")

	return cmd
}

func createSecret(cmd *cobra.Command, secretType string) error {
	ctx := context.Background()

	// Проверяем аутентификацию
	auth, err := store.GetAuth(ctx)
	if err != nil {
		return fmt.Errorf("failed to get auth: %w", err)
	}
	if auth == nil {
		return fmt.Errorf("not logged in. Please run 'gophkeeper auth login' first")
	}

	// Получаем общие параметры
	title, _ := cmd.Flags().GetString("title")
	meta, _ := cmd.Flags().GetString("meta")

	// Определяем тип секрета
	var secretTypeInt client.SecretType
	var data []byte
	//var metaMap map[string]interface{}

	switch secretType {
	case "password", "credentials":
		secretTypeInt = client.SecretTypeCredentials
		data, err = createCredentialsData(cmd)
	case "card", "bank_card":
		secretTypeInt = client.SecretTypeBankCard
		data, err = createBankCardData(cmd)
	case "text":
		secretTypeInt = client.SecretTypeText
		data, err = createTextData(cmd)
	case "binary":
		secretTypeInt = client.SecretTypeBinary
		data, err = createBinaryData(cmd)
	default:
		return fmt.Errorf("unknown secret type: %s. Supported: password, card, text, binary", secretType)
	}

	if err != nil {
		return err
	}

	// Создаём секрет
	secret := &client.Secret{
		ID:            uuid.New().String(),
		Type:          secretTypeInt,
		Title:         title,
		EncryptedData: data, // TODO: зашифровать данные мастер-ключом
		Meta:          meta,
		Version:       1,
		CreatedAt:     time.Now(),
		IsDeleted:     false,
	}

	// Сохраняем локально
	if err := store.SaveSecrets(ctx, []*client.Secret{secret}); err != nil {
		return fmt.Errorf("failed to save secret locally: %w", err)
	}

	// Отправляем на сервер
	authCtx := grpcClient.WithAuth(ctx)
	resp, err := grpcClient.GetKeeperClient().CreateSecret(authCtx, grpc.SecretToProtoSecretCreate(secret))
	if err != nil {
		return fmt.Errorf("failed to create secret on server: %w", err)
	}

	// // Обновляем ID из ответа сервера
	// secret.ID = resp.GetId()
	// if err := store.SaveSecrets(ctx, []*client.Secret{secret}); err != nil {
	// 	return fmt.Errorf("failed to update secret with server ID: %w", err)
	// }

	fmt.Printf("Secret created successfully!\n")
	fmt.Printf("ID: %s\n", resp.GetId())

	return nil
}

func createCredentialsData(cmd *cobra.Command) ([]byte, error) {
	login, _ := cmd.Flags().GetString("login")
	password, _ := cmd.Flags().GetString("password")
	url, _ := cmd.Flags().GetString("url")

	// Если пароль не указан в флаге, запрашиваем интерактивно
	if password == "" {
		fmt.Print("Enter password: ")
		passBytes, err := term.ReadPassword(int(syscall.Stdin))
		fmt.Println()
		if err != nil {
			return nil, err
		}
		password = string(passBytes)
	}

	data := client.CredentialsData{
		Login:    login,
		Password: password,
		URL:      url,
	}

	return json.Marshal(data)
}

func createBankCardData(cmd *cobra.Command) ([]byte, error) {
	number, _ := cmd.Flags().GetString("number")
	holder, _ := cmd.Flags().GetString("holder")
	expiry, _ := cmd.Flags().GetString("expiry")
	cvv, _ := cmd.Flags().GetString("cvv")
	bank, _ := cmd.Flags().GetString("bank")

	if number == "" || holder == "" || expiry == "" {
		return nil, fmt.Errorf("card number, holder, and expiry are required")
	}

	data := client.BankCardData{
		CardNumber:  number,
		CardHolder:  holder,
		ExpiryMonth: strings.Split(expiry, "/")[0],
		ExpiryYear:  strings.Split(expiry, "/")[1],
		CVV:         cvv,
		BankName:    bank,
	}

	return json.Marshal(data)
}

func createTextData(cmd *cobra.Command) ([]byte, error) {
	content, _ := cmd.Flags().GetString("content")

	if content == "" {
		fmt.Println("Enter text content (Ctrl+D to finish):")
		// Читаем многострочный ввод
		var builder strings.Builder
		scanner := bufio.NewScanner(os.Stdin)
		for scanner.Scan() {
			builder.WriteString(scanner.Text())
			builder.WriteString("\n")
		}
		content = builder.String()
	}

	data := client.TextData{
		Content: content,
	}

	return json.Marshal(data)
}

func createBinaryData(cmd *cobra.Command) ([]byte, error) {
	filePath, _ := cmd.Flags().GetString("file")

	if filePath == "" {
		return nil, fmt.Errorf("file path is required for binary type")
	}

	// Читаем файл
	fileData, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read file: %w", err)
	}

	data := client.BinaryData{
		Filename:    filepath.Base(filePath),
		ContentType: "application/octet-stream",
		Size:        int64(len(fileData)),
	}

	// Возвращаем сами бинарные данные (они будут зашифрованы)
	return json.Marshal(data)
}
