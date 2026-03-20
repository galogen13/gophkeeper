package commands

import (
	"context"
	"fmt"

	"github.com/galogen13/gophkeeper/internal/client"
	"github.com/galogen13/gophkeeper/internal/client/grpc"
	"github.com/spf13/cobra"
)

func NewSecretsDeleteCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "delete [id]",
		Short: "Delete a secret",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return deleteSecret(args[0])
		},
	}

	cmd.Flags().BoolP("permanent", "p", false, "Permanently delete (cannot be undone)")

	return cmd
}

func deleteSecret(id string) error {
	ctx := context.Background()

	// Проверяем аутентификацию
	auth, err := store.GetAuth(ctx)
	if err != nil {
		return fmt.Errorf("failed to get auth: %w", err)
	}
	if auth == nil {
		return fmt.Errorf("not logged in")
	}

	// Подтверждение
	fmt.Printf("Are you sure you want to delete secret %s? (y/N): ", id)
	var response string
	fmt.Scanln(&response)

	if response != "y" && response != "Y" {
		fmt.Println("Cancelled")
		return nil
	}

	// Помечаем как удалённое локально
	if err := store.DeleteSecret(ctx, id); err != nil {
		return fmt.Errorf("failed to delete secret locally: %w", err)
	}

	// Удаляем на сервере
	delSecret := &client.DeleteSecret{ID: id}
	authCtx := grpcClient.WithAuth(ctx)
	_, err = grpcClient.GetKeeperClient().DeleteSecret(authCtx, grpc.DeleteSecretToProtoDeleteSecret(delSecret))
	if err != nil {
		return fmt.Errorf("failed to delete secret on server: %w", err)
	}

	fmt.Printf("Secret %s deleted successfully\n", id)

	return nil
}
