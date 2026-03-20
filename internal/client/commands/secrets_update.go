package commands

import (
	"context"
	"fmt"

	"github.com/galogen13/gophkeeper/internal/client"
	"github.com/galogen13/gophkeeper/internal/client/grpc"
	"github.com/spf13/cobra"
)

func NewSecretsUpdateCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "update [id]",
		Short: "Update an existing secret",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return updateSecret(cmd, args[0])
		},
	}

	// Флаги для обновления
	cmd.Flags().String("title", "", "New title")
	cmd.Flags().String("login", "", "New login")
	cmd.Flags().String("password", "", "New password")
	cmd.Flags().String("url", "", "New URL")
	cmd.Flags().String("meta", "", "New metadata")

	return cmd
}

func updateSecret(cmd *cobra.Command, id string) error {
	ctx := context.Background()

	// Проверяем аутентификацию
	auth, err := store.GetAuth(ctx)
	if err != nil {
		return fmt.Errorf("failed to get auth: %w", err)
	}
	if auth == nil {
		return fmt.Errorf("not logged in")
	}

	// Получаем существующий секрет
	secret, err := store.GetSecret(ctx, id)
	if err != nil {
		return fmt.Errorf("failed to get secret: %w", err)
	}

	// Обновляем поля, если они указаны
	if title, _ := cmd.Flags().GetString("title"); title != "" {
		secret.Title = title
	}

	// Для обновления данных нужно расшифровать, изменить и заново зашифровать
	// Пока просто увеличиваем версию
	secret.Version++

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

	fmt.Printf("Secret %s updated successfully\n", id)

	return nil
}
