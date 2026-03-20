package commands

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"text/tabwriter"

	"github.com/galogen13/gophkeeper/internal/client"
	"github.com/spf13/cobra"
)

func NewSecretsGetCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "get [id]",
		Short: "Show secret details",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return getSecret(args[0])
		},
	}
}

func getSecret(id string) error {
	ctx := context.Background()

	// Проверяем аутентификацию
	auth, err := store.GetAuth(ctx)
	if err != nil {
		return fmt.Errorf("failed to get auth: %w", err)
	}
	if auth == nil {
		return fmt.Errorf("not logged in")
	}

	// Получаем секрет из локального хранилища
	secret, err := store.GetSecret(ctx, id)
	if err != nil {
		return fmt.Errorf("failed to get secret: %w", err)
	}

	// Выводим информацию
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)

	fmt.Fprintf(w, "ID:\t%s\n", secret.ID)
	fmt.Fprintf(w, "Title:\t%s\n", secret.Title)
	fmt.Fprintf(w, "Type:\t%s\n", secret.Type.String())
	fmt.Fprintf(w, "Version:\t%d\n", secret.Version)
	fmt.Fprintf(w, "Created:\t%s\n", secret.CreatedAt.Format("2006-01-02 15:04:05"))
	if secret.UpdatedAt != nil {
		fmt.Fprintf(w, "Updated:\t%s\n", secret.UpdatedAt.Format("2006-01-02 15:04:05"))
	}

	// Расшифровываем и показываем данные в зависимости от типа
	switch secret.Type {
	case client.SecretTypeCredentials:
		var data client.CredentialsData
		if err := json.Unmarshal(secret.EncryptedData, &data); err == nil {
			fmt.Fprintf(w, "\nCredentials:\n")
			fmt.Fprintf(w, "  Login:\t%s\n", data.Login)
			fmt.Fprintf(w, "  Password:\t%s\n", data.Password)
			if data.URL != "" {
				fmt.Fprintf(w, "  URL:\t%s\n", data.URL)
			}
		}
	case client.SecretTypeBankCard:
		var data client.BankCardData
		if err := json.Unmarshal(secret.EncryptedData, &data); err == nil {
			fmt.Fprintf(w, "\nBank Card:\n")
			fmt.Fprintf(w, "  Number:\t%s\n", maskCardNumber(data.CardNumber))
			fmt.Fprintf(w, "  Holder:\t%s\n", data.CardHolder)
			fmt.Fprintf(w, "  Expiry:\t%s/%s\n", data.ExpiryMonth, data.ExpiryYear)
			if data.CVV != "" {
				fmt.Fprintf(w, "  CVV:\t%s\n", "***")
			}
		}
	case client.SecretTypeText:
		var data client.TextData
		if err := json.Unmarshal(secret.EncryptedData, &data); err == nil {
			fmt.Fprintf(w, "\nContent:\n%s\n", data.Content)
		}
	case client.SecretTypeBinary:
		var data client.BinaryData
		if err := json.Unmarshal(secret.EncryptedData, &data); err == nil {
			fmt.Fprintf(w, "\nBinary file:\n")
			fmt.Fprintf(w, "  Filename:\t%s\n", data.Filename)
			fmt.Fprintf(w, "  Size:\t%d bytes\n", data.Size)
		}
	}

	w.Flush()

	return nil
}

func maskCardNumber(number string) string {
	if len(number) < 4 {
		return number
	}
	return "**** **** **** " + number[len(number)-4:]
}
