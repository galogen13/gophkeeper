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

	"github.com/galogen13/gophkeeper/internal/client"
	"github.com/galogen13/gophkeeper/internal/client/grpc"
	"github.com/galogen13/gophkeeper/internal/client/storage"
	"github.com/google/uuid"
	"golang.org/x/term"
)

// RunInteractiveMode запускает интерактивную оболочку
func RunInteractiveMode(version, date string) error {
	// Инициализируем клиент (без аутентификации)
	if err := initializeClient(); err != nil {
		return fmt.Errorf("failed to initialize: %w", err)
	}
	defer closeClient()

	fmt.Println("╔══════════════════════════════════════════╗")
	fmt.Println("║     GophKeeper - Secure Password Manager ║")
	fmt.Println("╚══════════════════════════════════════════╝")
	fmt.Println()

	// Аутентификация
	if err := authenticate(); err != nil {
		return fmt.Errorf("authentication failed: %w", err)
	}

	fmt.Println()
	fmt.Println("Welcome to GophKeeper!")
	fmt.Println("Type 'help' for available commands, 'exit' to quit")
	fmt.Println()

	// Интерактивный цикл
	scanner := bufio.NewScanner(os.Stdin)
	for {
		fmt.Print("gophkeeper> ")
		if !scanner.Scan() {
			break
		}

		input := strings.TrimSpace(scanner.Text())
		if input == "" {
			continue
		}

		// Обработка команды
		if err := executeCommand(input); err != nil {
			fmt.Printf("Error: %v\n", err)
		}
	}

	return nil
}

// authenticate запрашивает учётные данные и аутентифицирует пользователя
func authenticate() error {
	reader := bufio.NewReader(os.Stdin)

	fmt.Print("Email: ")
	email, err := reader.ReadString('\n')
	if err != nil {
		return err
	}
	email = strings.TrimSpace(email)

	fmt.Print("Password: ")
	password, err := term.ReadPassword(int(syscall.Stdin))
	fmt.Println()
	if err != nil {
		return err
	}

	fmt.Print("Master password (for encryption): ")
	masterPass, err := term.ReadPassword(int(syscall.Stdin))
	fmt.Println()
	if err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	registerLoginInfo := &client.RegisterLoginInfo{
		Email:    email,
		Password: string(password),
	}

	// Проверяем, существует ли уже аккаунт (по наличию соли)
	if keyManager.HasMasterKey() {
		resp, err := grpcClient.GetAuthClient().Login(ctx, grpc.RegisterInfoToProtoLoginRequest(registerLoginInfo))
		if err != nil {
			return fmt.Errorf("login failed: %w", err)
		}

		if err := keyManager.LoadMasterKey(string(masterPass)); err != nil {
			return fmt.Errorf("master key error: %w", err)
		}

		auth := grpc.AuthFromProtoAuth(resp, "")
		if err := store.SaveAuth(ctx, auth); err != nil {
			return err
		}

		grpcClient.SetToken(auth.AccessToken)

		return syncDataInteractive()

	} else {
		// Регистрация нового пользователя
		fmt.Print("Confirm master password: ")
		confirmPass, err := term.ReadPassword(int(syscall.Stdin))
		fmt.Println()
		if err != nil {
			return err
		}

		if string(masterPass) != string(confirmPass) {
			return fmt.Errorf("master passwords do not match")
		}

		resp, err := grpcClient.GetAuthClient().
			Register(ctx, grpc.RegisterInfoToProtoRegisterRequest(registerLoginInfo))
		if err != nil {
			return fmt.Errorf("registration failed: %w", err)
		}

		if err := keyManager.InitMasterKey(string(masterPass)); err != nil {
			return fmt.Errorf("failed to init master key: %w", err)
		}

		auth := grpc.AuthFromProtoAuth(resp, "")
		if err := store.SaveAuth(ctx, auth); err != nil {
			return err
		}

		grpcClient.SetToken(auth.AccessToken)
	}

	return nil
}

// executeCommand выполняет команду в интерактивном режиме
func executeCommand(input string) error {
	args := strings.Fields(input)
	if len(args) == 0 {
		return nil
	}

	command := args[0]
	cmdArgs := args[1:]

	switch command {
	case "exit", "quit":
		fmt.Println("Goodbye!")
		os.Exit(0)
		return nil

	case "help":
		printHelp()
		return nil

	case "list", "ls":
		return listSecretsInteractive()

	case "get":
		if len(cmdArgs) < 1 {
			return fmt.Errorf("usage: get <id>")
		}
		return getSecretInteractive(cmdArgs[0])

	case "create":
		if len(cmdArgs) < 2 {
			return fmt.Errorf("usage: create <type> --title <title> [options]")
		}
		return createSecretInteractive(cmdArgs)

	case "delete", "rm":
		if len(cmdArgs) < 1 {
			return fmt.Errorf("usage: delete <id>")
		}
		return deleteSecretInteractive(cmdArgs[0])

	case "sync":
		return syncDataInteractive()

	case "status":
		return statusInteractive()

	case "logout":
		return logoutInteractive()

	case "login":
		return authenticate()

	default:
		return fmt.Errorf("unknown command: %s. Type 'help' for available commands", command)
	}
}

func printHelp() {
	fmt.Println(`
Available commands:
  help                Show this help message
  exit, quit          Exit GophKeeper

  list, ls            List all secrets
  get <id>            Show secret details
  create <type>       Create a new secret (password/card/text/binary)
  delete <id>         Delete a secret
  sync                Synchronize with server
  status              Show authentication status
  login               Login and start session
  logout              Logout and clear session`)
}

// Интерактивные версии команд (используют уже загруженный мастер-ключ)

func listSecretsInteractive() error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	secrets, err := store.ListActiveSecrets(ctx)
	if err != nil {
		return err
	}

	if len(secrets) == 0 {
		fmt.Println("No secrets found. Create one with 'create'")
		return nil
	}

	fmt.Printf("\n%-36s %-20s %-12s %-20s\n", "ID", "TITLE", "TYPE", "CREATED")
	fmt.Println(strings.Repeat("-", 88))

	for _, s := range secrets {
		fmt.Printf("%-36s %-20s %-12s %-20s\n",
			s.ID,
			truncate(s.Title, 20),
			s.Type.String(),
			s.CreatedAt.Format("2006-01-02 15:04"),
		)
	}
	fmt.Println()
	return nil
}

func getSecretInteractive(id string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	mk := keyManager.GetMasterKey()
	if mk == nil {
		return fmt.Errorf("master key not loaded")
	}

	secret, err := store.GetSecret(ctx, id)
	if err != nil {
		return err
	}

	decryptedData, err := mk.Decrypt(secret.EncryptedData)
	if err != nil {
		return fmt.Errorf("failed to decrypt: %w", err)
	}

	fmt.Printf("\nID: %s\n", secret.ID)
	fmt.Printf("Title: %s\n", secret.Title)
	fmt.Printf("Type: %s\n", secret.Type.String())
	fmt.Printf("Created: %s\n", secret.CreatedAt.Format("2006-01-02 15:04:05"))
	if secret.UpdatedAt != nil {
		fmt.Printf("Updated: %s\n", secret.UpdatedAt.Format("2006-01-02 15:04:05"))
	}
	fmt.Println()

	switch secret.Type {
	case client.SecretTypeCredentials:
		var data client.CredentialsData
		if err := json.Unmarshal(decryptedData, &data); err != nil {
			return fmt.Errorf("failed to parse credentials: %w", err)
		}
		fmt.Println("Credentials:")
		fmt.Printf("  Login:    %s\n", data.Login)
		fmt.Printf("  Password: %s\n", data.Password)
		if data.URL != "" {
			fmt.Printf("  URL:      %s\n", data.URL)
		}

	case client.SecretTypeBankCard:
		var data client.BankCardData
		if err := json.Unmarshal(decryptedData, &data); err != nil {
			return fmt.Errorf("failed to parse card data: %w", err)
		}
		fmt.Println("Bank Card:")
		fmt.Printf("  Number: %s\n", data.CardNumber)
		fmt.Printf("  Holder: %s\n", data.CardHolder)
		fmt.Printf("  Expiry: %s/%s\n", data.ExpiryMonth, data.ExpiryYear)
		fmt.Printf("  CVV: %s\n", data.CVV)
		if data.BankName != "" {
			fmt.Printf("  Bank:   %s\n", data.BankName)
		}

	case client.SecretTypeText:
		var data client.TextData
		if err := json.Unmarshal(decryptedData, &data); err != nil {
			return fmt.Errorf("failed to parse text data: %w", err)
		}
		fmt.Println("Content:")
		fmt.Println(strings.Repeat("-", 40))
		fmt.Println(data.Content)
		fmt.Println(strings.Repeat("-", 40))

	case client.SecretTypeBinary:
		var data client.BinaryData
		if err := json.Unmarshal([]byte(secret.Meta), &data); err != nil {
			return fmt.Errorf("failed to parse binary metadata: %w", err)
		}
		fmt.Println("Binary File:")
		fmt.Printf("  Filename: %s\n", data.Filename)
		fmt.Printf("  Size:     %d bytes\n", data.Size)

		fmt.Println()
		fmt.Println("Enter the directory path to save the file to this directory. Or press Enter to skip this step.")
		fmt.Println("Directory path:")

		reader := bufio.NewReader(os.Stdin)

		dirPath, err := reader.ReadString('\n')
		if err != nil {
			return fmt.Errorf("failed to get directory path: %w", err)
		}
		dirPath = strings.TrimSpace(dirPath)
		if dirPath != "" {

			if _, err := os.Stat(dirPath); os.IsNotExist(err) {
				fmt.Printf("Directory does not exist. Create it? (y/n): ")
				answer, _ := reader.ReadString('\n')
				answer = strings.TrimSpace(strings.ToLower(answer))

				if answer == "y" || answer == "yes" {
					if err := os.MkdirAll(dirPath, 0755); err != nil {
						return fmt.Errorf("failed to create directory: %w", err)
					}
				} else {
					fmt.Println("File not saved.")
					break
				}
			}

			filePath := filepath.Join(dirPath, data.Filename)

			if err := os.WriteFile(filePath, decryptedData, 0644); err != nil {
				return fmt.Errorf("failed to save file: %w", err)
			}

			fmt.Printf("File saved to: %s\n", filePath)
		} else {
			fmt.Println("File not saved.")
		}
	}

	fmt.Println()
	return nil
}

func createSecretInteractive(args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("usage: create <type> [options]")
	}

	secretType := args[0]
	mk := keyManager.GetMasterKey()
	if mk == nil {
		return fmt.Errorf("master key not loaded")
	}

	// Парсим флаги (упрощённый парсер для интерактивного режима)
	flags := parseFlags(args[1:])

	title := flags["title"]
	if title == "" {
		return fmt.Errorf("title is required (--title)")
	}

	var data []byte
	var secretTypeInt client.SecretType

	switch secretType {
	case "password", "credentials":
		secretTypeInt = client.SecretTypeCredentials
		credData, err := collectCredentialsData(flags)
		if err != nil {
			return err
		}
		data, err = json.Marshal(credData)
		if err != nil {
			return err
		}

	case "card", "bank_card":
		secretTypeInt = client.SecretTypeBankCard
		cardData, err := collectBankCardData(flags)
		if err != nil {
			return err
		}
		data, err = json.Marshal(cardData)
		if err != nil {
			return err
		}

	case "text":
		secretTypeInt = client.SecretTypeText
		textData, err := collectTextData(flags)
		if err != nil {
			return err
		}
		data, err = json.Marshal(textData)
		if err != nil {
			return err
		}

	case "binary":
		secretTypeInt = client.SecretTypeBinary
		binaryData, fileContent, err := collectBinaryData(flags)
		if err != nil {
			return err
		}
		// Для бинарных данных: metadata хранится в JSON, а сами данные — в EncryptedData
		metaData, err := json.Marshal(binaryData)
		if err != nil {
			return err
		}
		data = fileContent
		// Сохраняем метаданные отдельно
		flags["meta"] = string(metaData)

	default:
		return fmt.Errorf("unknown type: %s. Supported: password, card, text, binary", secretType)
	}

	encryptedData, err := mk.Encrypt(data)
	if err != nil {
		return err
	}

	secret := &client.Secret{
		ID:            uuid.New().String(),
		Type:          secretTypeInt,
		Title:         title,
		EncryptedData: encryptedData,
		Meta:          flags["meta"],
		CreatedAt:     time.Now(),
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Отправляем сначала на сервер
	authCtx := grpcClient.WithAuth(ctx)
	resp, err := grpcClient.GetKeeperClient().CreateSecret(authCtx, grpc.SecretToProtoSecretCreate(secret))
	if err != nil {
		return fmt.Errorf("failed to create secret on server: %w", err)
	}

	if err := store.SaveSecrets(ctx, []*client.Secret{secret}); err != nil {
		return err
	}

	fmt.Printf("Secret created: %s\n", resp.GetId())
	return nil
}

func deleteSecretInteractive(id string) error {

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	authCtx := grpcClient.WithAuth(ctx)

	delSecret := &client.DeleteSecret{ID: id}
	_, err := grpcClient.GetKeeperClient().DeleteSecret(authCtx, grpc.DeleteSecretToProtoDeleteSecret(delSecret))
	if err != nil {
		return fmt.Errorf("failed to delete secret on server: %w", err)
	}

	if err := store.DeleteSecret(ctx, id); err != nil {
		return err
	}

	fmt.Printf("Secret %s deleted\n", id)
	return nil
}

func syncDataInteractive() error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Получаем время последней синхронизации
	lastSync, err := store.GetLastSyncTime(ctx)
	if err != nil {
		return fmt.Errorf("sync failed, getting last sync time failed: %w", err)
	}

	// Создаём контекст с токеном
	authCtx := grpcClient.WithAuth(ctx)

	syncInfo := &client.SyncInfo{LastSync: lastSync}

	resp, err := grpcClient.GetKeeperClient().SyncSecrets(authCtx, grpc.SyncInfoToProtoSyncRequest(syncInfo))
	if err != nil {
		return fmt.Errorf("sync failed: %w", err)
	}

	// mk := keyManager.GetMasterKey()
	secrets := make([]*client.Secret, 0, len(resp.GetSecrets()))
	for _, pbSecret := range resp.GetSecrets() {
		secret := grpc.SecretFromProtoSecret(pbSecret)
		secrets = append(secrets, secret)
	}
	if err := store.SaveSecrets(ctx, secrets); err != nil {
		return fmt.Errorf("failed to save secrets during synchronization: %w", err)
	}

	for _, id := range resp.GetDeletedIds() {
		if err := store.DeleteSecret(ctx, id); err != nil {
			// Игнорируем ошибки, если секрет уже удалён локально
			if err != storage.ErrNotFound {
				return fmt.Errorf("failed to delete secret %s: %w", id, err)
			}
		}
	}

	store.SetLastSyncTime(ctx, time.Now())
	fmt.Printf("Synced: %d new/updated, %d deleted\n", len(resp.GetSecrets()), len(resp.GetDeletedIds()))
	return nil
}

func statusInteractive() error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	auth, err := store.GetAuth(ctx)
	if err != nil {
		return err
	}

	if auth == nil {
		fmt.Println("Not authenticated")
		return nil
	}

	if auth.IsExpired() {
		fmt.Println("Session expired")
	} else {
		fmt.Println("Authenticated")
		expiresIn := time.Until(auth.SavedAt.Add(time.Duration(auth.ExpiresIn) * time.Second))
		fmt.Printf("Session expires in: %v\n", expiresIn.Round(time.Second))
	}

	if keyManager.GetMasterKey() != nil {
		fmt.Println("Master key: loaded (in memory)")
	} else {
		fmt.Println("Master key: not loaded")
	}

	return nil
}

func logoutInteractive() error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	store.ClearAuth(ctx)
	grpcClient.SetToken("")
	keyManager.Clear()
	fmt.Println("Logged out")
	return nil
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n-3] + "..."
}

// collectCredentialsData собирает данные для логина/пароля
func collectCredentialsData(flags map[string]string) (client.CredentialsData, error) {
	var data client.CredentialsData

	if login, ok := flags["login"]; ok {
		data.Login = login
	} else {
		fmt.Print("Login: ")
		fmt.Scanln(&data.Login)
	}

	if pass, ok := flags["password"]; ok {
		data.Password = pass
	} else {
		fmt.Print("Password: ")
		passBytes, err := term.ReadPassword(int(syscall.Stdin))
		fmt.Println()
		if err != nil {
			return data, err
		}
		data.Password = string(passBytes)
	}

	if url, ok := flags["url"]; ok {
		data.URL = url
	} else {
		fmt.Print("URL (optional): ")
		fmt.Scanln(&data.URL)
	}

	return data, nil
}

// collectBankCardData собирает данные банковской карты
func collectBankCardData(flags map[string]string) (client.BankCardData, error) {
	var data client.BankCardData

	if num, ok := flags["number"]; ok {
		data.CardNumber = num
	} else {
		fmt.Print("Card number: ")
		fmt.Scanln(&data.CardNumber)
	}

	if holder, ok := flags["holder"]; ok {
		data.CardHolder = holder
	} else {
		fmt.Print("Card holder name: ")
		fmt.Scanln(&data.CardHolder)
	}

	if expiry, ok := flags["expiry"]; ok {
		parts := strings.Split(expiry, "/")
		if len(parts) == 2 {
			data.ExpiryMonth = parts[0]
			data.ExpiryYear = parts[1]
		}
	} else {
		fmt.Print("Expiry (MM/YY): ")
		var expiry string
		fmt.Scanln(&expiry)
		parts := strings.Split(expiry, "/")
		if len(parts) == 2 {
			data.ExpiryMonth = parts[0]
			data.ExpiryYear = parts[1]
		}
	}

	if cvv, ok := flags["cvv"]; ok {
		data.CVV = cvv
	} else {
		fmt.Print("CVV (optional): ")
		fmt.Scanln(&data.CVV)
	}

	if bank, ok := flags["bank"]; ok {
		data.BankName = bank
	} else {
		fmt.Print("Bank name (optional): ")
		fmt.Scanln(&data.BankName)
	}

	return data, nil
}

// collectTextData собирает текстовые данные
func collectTextData(flags map[string]string) (client.TextData, error) {
	var data client.TextData

	if content, ok := flags["content"]; ok {
		data.Content = content
		return data, nil
	}

	fmt.Println("Enter content (multi-line, type 'END' on a new line to finish):")

	var lines []string
	scanner := bufio.NewScanner(os.Stdin)

	for scanner.Scan() {
		line := scanner.Text()
		if line == "END" {
			break
		}
		lines = append(lines, line)
	}

	if err := scanner.Err(); err != nil {
		return data, err
	}

	data.Content = strings.Join(lines, "\n")
	return data, nil
}

// collectBinaryData собирает бинарные данные из файла
func collectBinaryData(flags map[string]string) (client.BinaryData, []byte, error) {
	var data client.BinaryData
	var filePath string

	if path, ok := flags["file"]; ok {
		filePath = path
	} else {
		fmt.Print("File path: ")
		fmt.Scanln(&filePath)
	}

	if filePath == "" {
		return data, nil, fmt.Errorf("file path is required")
	}

	fileContent, err := os.ReadFile(filePath)
	if err != nil {
		return data, nil, fmt.Errorf("failed to read file: %w", err)
	}

	data.Filename = filepath.Base(filePath)
	data.Size = int64(len(fileContent))

	return data, fileContent, nil
}

// Вспомогательные функции
func parseFlags(args []string) map[string]string {
	flags := make(map[string]string)
	for i := 0; i < len(args); i++ {
		if strings.HasPrefix(args[i], "--") {
			key := strings.TrimPrefix(args[i], "--")
			if i+1 < len(args) && !strings.HasPrefix(args[i+1], "--") {
				flags[key] = args[i+1]
				i++
			} else {
				flags[key] = "true"
			}
		}
	}
	return flags
}
